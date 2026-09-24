package sysinfo

import (
	"bufio"
	"encoding/hex"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	psnet "github.com/shirou/gopsutil/v4/net"
)

// NetInterface is one entry of si.networkInterfaces().
type NetInterface struct {
	Iface     string `json:"iface"`
	IfaceName string `json:"ifaceName"`
	Default   bool   `json:"default"`
	IP4       string `json:"ip4"`
	IP4Subnet string `json:"ip4subnet"`
	IP6       string `json:"ip6"`
	MAC       string `json:"mac"`
	Internal  bool   `json:"internal"`
	Operstate string `json:"operstate"`
}

// NetworkInterfaces lists the network interfaces. The interface holding the
// default route is listed first, so that automatic detection picks it over
// virtual bridges.
func NetworkInterfaces() ([]NetInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	def := defaultInterface()
	list := make([]NetInterface, 0, len(ifaces))
	for _, ifc := range ifaces {
		e := NetInterface{
			Iface:     ifc.Name,
			IfaceName: ifc.Name,
			MAC:       ifc.HardwareAddr.String(),
			Internal:  ifc.Flags&net.FlagLoopback != 0,
			Operstate: operstate(ifc),
			Default:   ifc.Name == def,
		}
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				if e.IP4 == "" {
					e.IP4 = ip4.String()
					e.IP4Subnet = net.IP(ipnet.Mask).String()
				}
			} else if e.IP6 == "" {
				e.IP6 = ipnet.IP.String()
			}
		}
		if e.Default {
			list = append([]NetInterface{e}, list...)
		} else {
			list = append(list, e)
		}
	}
	return list, nil
}

func operstate(ifc net.Interface) string {
	if runtime.GOOS == "linux" {
		if data, err := os.ReadFile("/sys/class/net/" + ifc.Name + "/operstate"); err == nil {
			st := strings.TrimSpace(string(data))
			// Tunnels and some Wi-Fi drivers report "unknown" while working.
			if st == "unknown" && ifc.Flags&net.FlagUp != 0 && ifc.Flags&net.FlagRunning != 0 {
				return "up"
			}
			return st
		}
	}
	if ifc.Flags&net.FlagUp != 0 && ifc.Flags&net.FlagRunning != 0 {
		return "up"
	}
	return "down"
}

// defaultInterface finds the interface used to reach the Internet. Dialing
// UDP does not send any packet, it only resolves the route.
func defaultInterface() string {
	conn, err := net.DialTimeout("udp4", "1.1.1.1:80", time.Second)
	if err != nil {
		return ""
	}
	defer conn.Close()
	local, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return ""
	}
	ifaces, _ := net.Interfaces()
	for _, ifc := range ifaces {
		addrs, _ := ifc.Addrs()
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.Equal(local.IP) {
				return ifc.Name
			}
		}
	}
	return ""
}

// NetStats is one entry of si.networkStats().
type NetStats struct {
	Iface     string  `json:"iface"`
	Operstate string  `json:"operstate"`
	RxBytes   uint64  `json:"rx_bytes"`
	TxBytes   uint64  `json:"tx_bytes"`
	RxSec     float64 `json:"rx_sec"`
	TxSec     float64 `json:"tx_sec"`
	Ms        int64   `json:"ms"`
}

type netSample struct {
	rx, tx uint64
	at     time.Time
}

// NetworkStats returns traffic counters and rates (bytes per second) for
// iface, computed against the previous call.
func (s *SI) NetworkStats(iface string) ([]NetStats, error) {
	counters, err := psnet.IOCounters(true)
	if err != nil {
		return nil, err
	}
	s.netMu.Lock()
	defer s.netMu.Unlock()
	now := time.Now()
	for _, c := range counters {
		if c.Name != iface {
			continue
		}
		st := NetStats{Iface: c.Name, Operstate: "up", RxBytes: c.BytesRecv, TxBytes: c.BytesSent}
		if prev, ok := s.netPrev[iface]; ok {
			elapsed := now.Sub(prev.at)
			if secs := elapsed.Seconds(); secs > 0 && c.BytesRecv >= prev.rx && c.BytesSent >= prev.tx {
				st.RxSec = float64(c.BytesRecv-prev.rx) / secs
				st.TxSec = float64(c.BytesSent-prev.tx) / secs
				st.Ms = elapsed.Milliseconds()
			}
		}
		s.netPrev[iface] = netSample{rx: c.BytesRecv, tx: c.BytesSent, at: now}
		return []NetStats{st}, nil
	}
	return []NetStats{{Iface: iface, Operstate: "down"}}, nil
}

// Connection is one entry of si.networkConnections(). The lowercase
// peeraddress duplicate is the field name read by the original globe module.
type Connection struct {
	Protocol     string `json:"protocol"`
	LocalAddress string `json:"localAddress"`
	LocalPort    string `json:"localPort"`
	PeerAddress  string `json:"peerAddress"`
	PeerAddr     string `json:"peeraddress"`
	PeerPort     string `json:"peerPort"`
	State        string `json:"state"`
}

var tcpStates = map[string]string{
	"01": "ESTABLISHED", "02": "SYN_SENT", "03": "SYN_RECV", "04": "FIN_WAIT1",
	"05": "FIN_WAIT2", "06": "TIME_WAIT", "07": "CLOSE", "08": "CLOSE_WAIT",
	"09": "LAST_ACK", "0A": "LISTEN", "0B": "CLOSING",
}

// NetworkConnections lists the TCP connections.
func NetworkConnections() ([]Connection, error) {
	if runtime.GOOS == "linux" {
		// Reading /proc/net directly avoids scanning every process for
		// socket ownership, which gopsutil does.
		conns := parseProcNet("/proc/net/tcp", "tcp", false)
		return append(conns, parseProcNet("/proc/net/tcp6", "tcp6", true)...), nil
	}
	stats, err := psnet.Connections("tcp")
	if err != nil {
		return nil, err
	}
	conns := make([]Connection, 0, len(stats))
	for _, c := range stats {
		conns = append(conns, Connection{
			Protocol:     "tcp",
			LocalAddress: c.Laddr.IP,
			LocalPort:    strconv.Itoa(int(c.Laddr.Port)),
			PeerAddress:  c.Raddr.IP,
			PeerAddr:     c.Raddr.IP,
			PeerPort:     strconv.Itoa(int(c.Raddr.Port)),
			State:        c.Status,
		})
	}
	return conns, nil
}

func parseProcNet(path, proto string, v6 bool) []Connection {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var conns []Connection
	sc := bufio.NewScanner(f)
	sc.Scan() // header
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 {
			continue
		}
		lip, lport := parseHexAddr(fields[1], v6)
		rip, rport := parseHexAddr(fields[2], v6)
		conns = append(conns, Connection{
			Protocol:     proto,
			LocalAddress: lip,
			LocalPort:    lport,
			PeerAddress:  rip,
			PeerAddr:     rip,
			PeerPort:     rport,
			State:        tcpStates[fields[3]],
		})
	}
	return conns
}

// parseHexAddr decodes the "0100007F:0035" notation of /proc/net/tcp.
func parseHexAddr(s string, v6 bool) (string, string) {
	host, port, ok := strings.Cut(s, ":")
	if !ok {
		return "", ""
	}
	raw, err := hex.DecodeString(host)
	if err != nil {
		return "", ""
	}
	// The kernel prints each 32-bit word in host (little endian) order.
	for i := 0; i+4 <= len(raw); i += 4 {
		raw[i], raw[i+1], raw[i+2], raw[i+3] = raw[i+3], raw[i+2], raw[i+1], raw[i]
	}
	ip := net.IP(raw)
	if v6 {
		if v4 := ip.To4(); v4 != nil {
			ip = v4
		}
	}
	p, _ := strconv.ParseUint(port, 16, 16)
	return ip.String(), strconv.FormatUint(p, 10)
}
