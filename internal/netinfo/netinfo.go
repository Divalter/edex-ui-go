// Package netinfo implements the network probes of eDEX-UI's netstat and
// globe modules: external IP discovery, TCP "ping" and GeoIP lookups.
package netinfo

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

// Location is the subset of a GeoLite2 City record used by the UI.
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GeoResult mirrors the shape of maxmind's lookup result ({location: ...}).
type GeoResult struct {
	Location Location `json:"location"`
}

// geoDBSources are tried in order. The first one is the redistribution used
// by the original eDEX-UI (geolite2-redist).
var geoDBSources = []string{
	"https://raw.githubusercontent.com/GitSquared/node-geolite2-redist/master/redist/GeoLite2-City.tar.gz",
	"https://github.com/P3TERX/GeoLite.mmdb/raw/download/GeoLite2-City.mmdb",
}

const geoDBMaxAge = 30 * 24 * time.Hour

// Service holds the GeoIP database.
type Service struct {
	cacheDir string
	client   *http.Client

	mu sync.RWMutex
	db *geoip2.Reader
}

// New creates the service. Call LoadGeoDB (typically in a goroutine) to
// download or open the GeoIP database.
func New(cacheDir string) *Service {
	return &Service{cacheDir: cacheDir, client: &http.Client{Timeout: 60 * time.Second}}
}

// LoadGeoDB opens the cached GeoLite2 City database, downloading it when
// missing or older than 30 days.
func (s *Service) LoadGeoDB() error {
	path := filepath.Join(s.cacheDir, "GeoLite2-City.mmdb")
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > geoDBMaxAge {
		if derr := s.download(path); derr != nil {
			if err != nil {
				return derr
			}
			log.Printf("GeoIP: update failed, using cached database: %v", derr)
		}
	}
	db, err := geoip2.Open(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	if s.db != nil {
		s.db.Close()
	}
	s.db = db
	s.mu.Unlock()
	return nil
}

func (s *Service) download(dst string) error {
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return err
	}
	var errs []error
	for _, url := range geoDBSources {
		if err := s.downloadFrom(url, dst); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", url, err))
			continue
		}
		return nil
	}
	return errors.Join(errs...)
}

func (s *Service) downloadFrom(url, dst string) error {
	resp, err := s.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var src io.Reader = resp.Body
	if strings.HasSuffix(url, ".tar.gz") {
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}
		tr := tar.NewReader(gz)
		for {
			h, err := tr.Next()
			if err != nil {
				return fmt.Errorf("no .mmdb in archive: %w", err)
			}
			if strings.HasSuffix(h.Name, ".mmdb") {
				src = tr
				break
			}
		}
	}
	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, src); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// Ready reports whether GeoIP lookups are available.
func (s *Service) Ready() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db != nil
}

// Lookup returns the location of ip, or nil when unknown.
func (s *Service) Lookup(ip string) *GeoResult {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return nil
	}
	rec, err := s.db.City(parsed)
	if err != nil || (rec.Location.Latitude == 0 && rec.Location.Longitude == 0) {
		return nil
	}
	return &GeoResult{Location: Location{Latitude: rec.Location.Latitude, Longitude: rec.Location.Longitude}}
}

// LookupAll resolves several addresses at once.
func (s *Service) LookupAll(ips []string) map[string]*GeoResult {
	res := make(map[string]*GeoResult, len(ips))
	for _, ip := range ips {
		res[ip] = s.Lookup(ip)
	}
	return res
}

// ExternalIP is the public address as seen by myexternalip.com, with its
// location when the GeoIP database is available.
type ExternalIP struct {
	IP  string    `json:"ip"`
	Geo *Location `json:"geo"`
}

func dialerFrom(localAddr string, timeout time.Duration) *net.Dialer {
	d := &net.Dialer{Timeout: timeout}
	if ip := net.ParseIP(localAddr); ip != nil {
		d.LocalAddr = &net.TCPAddr{IP: ip}
	}
	return d
}

// FetchExternalIP asks myexternalip.com for the public IP, going out through
// the interface that owns localAddr, like the original netstat module.
func (s *Service) FetchExternalIP(localAddr string) (*ExternalIP, error) {
	d := dialerFrom(localAddr, 5*time.Second)
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return d.DialContext(ctx, "tcp4", addr)
			},
			DisableKeepAlives: true,
		},
	}
	resp, err := client.Get("https://myexternalip.com/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var body struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body); err != nil {
		return nil, fmt.Errorf("parse myexternalip.com response: %w", err)
	}
	res := &ExternalIP{IP: body.IP}
	if geo := s.Lookup(body.IP); geo != nil {
		res.Geo = &geo.Location
	}
	return res, nil
}

// Ping measures the time to open a TCP connection to target:port from
// localAddr, in milliseconds (1.9s timeout, like the original).
func Ping(target string, port int, localAddr string) (float64, error) {
	d := dialerFrom(localAddr, 1900*time.Millisecond)
	start := time.Now()
	conn, err := d.Dial("tcp4", net.JoinHostPort(target, fmt.Sprint(port)))
	if err != nil {
		return 0, err
	}
	elapsed := time.Since(start)
	conn.Close()
	return float64(elapsed.Microseconds()) / 1000, nil
}
