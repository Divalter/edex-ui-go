// Package bridge connects the UI to the Go backend. It replaces what
// Electron provided to the original eDEX-UI: ipcRenderer/ipcMain (RPC calls
// and an event stream), the per-terminal WebSocket servers, and file://
// access to local files.
//
// The desktop app does not open any network port: it uses Dispatch through
// the Wails IPC, Wails events and ServeFile as a Wails asset handler. The
// HTTP server (Start) is only used by the browser-based development server
// (cmd/edex-serve); it listens on the loopback interface and every request
// must carry a random token.
package bridge

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Handler implements one RPC method. args are the JSON-encoded arguments.
type Handler func(args []json.RawMessage) (any, error)

// TTYHandler attaches a WebSocket client to the terminal identified by port.
type TTYHandler func(port int, w http.ResponseWriter, r *http.Request)

// Server is the bridge server.
type Server struct {
	Token string
	Port  int

	mu       sync.RWMutex
	handlers map[string]Handler
	tty      TTYHandler
	static   fs.FS

	clientsMu sync.Mutex
	clients   map[*eventClient]struct{}

	listener net.Listener
	srv      *http.Server
}

// New creates a server with a fresh random token.
func New() (*Server, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return &Server{
		Token:    hex.EncodeToString(b),
		handlers: map[string]Handler{},
		clients:  map[*eventClient]struct{}{},
	}, nil
}

// Handle registers an RPC method.
func (s *Server) Handle(name string, h Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[name] = h
}

// HandleTTY registers the terminal WebSocket handler.
func (s *Server) HandleTTY(h TTYHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tty = h
}

// ServeStatic serves the UI itself at "/" (used by the browser-based
// development server; in the desktop app Wails serves the UI).
func (s *Server) ServeStatic(fsys fs.FS) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.static = fsys
}

// Start listens on addr (e.g. "127.0.0.1:0") and serves in the background.
func (s *Server) Start(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = l
	s.Port = l.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/rpc/", s.auth(s.handleRPC))
	mux.HandleFunc("/events", s.auth(s.handleEvents))
	mux.HandleFunc("/tty/", s.auth(s.handleTTY))
	mux.HandleFunc("/file", s.auth(ServeFile))
	mux.HandleFunc("/", s.handleStatic)

	s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := s.srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("bridge: %v", err)
		}
	}()
	return nil
}

// Close stops the server.
func (s *Server) Close() error {
	if s.srv == nil {
		return nil
	}
	return s.srv.Close()
}

// sameOrigin reports whether a browser request comes from the page served
// by this server. Requests without an Origin header (non-browser clients,
// same-origin media loads) are accepted and still need the token.
func (s *Server) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return origin == s.URL() || origin == fmt.Sprintf("http://localhost:%d", s.Port)
}

// auth protects against the vulnerability of the original eDEX-UI, whose
// terminal WebSocket accepted any client: a website could connect to it and
// run commands (cross-site WebSocket hijacking). Every request needs the
// random token, and requests from other origins are rejected even with it.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.sameOrigin(r) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		token := r.Header.Get("X-Edex-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.Token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

type rpcResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/rpc/")
	var args []json.RawMessage
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<20))
	if err == nil && len(body) > 0 {
		err = json.Unmarshal(body, &args)
	}
	if err != nil {
		writeJSON(w, http.StatusBadRequest, rpcResponse{Error: "bad arguments: " + err.Error()})
		return
	}
	res, err := s.Dispatch(name, args)
	if err != nil {
		writeJSON(w, http.StatusOK, rpcResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rpcResponse{Result: res})
}

// Dispatch runs an RPC method. It backs both the HTTP endpoint and the
// Wails binding.
func (s *Server) Dispatch(name string, args []json.RawMessage) (any, error) {
	s.mu.RLock()
	h, ok := s.handlers[name]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown method %s", name)
	}
	return h(args)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleTTY(w http.ResponseWriter, r *http.Request) {
	port, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/tty/"))
	if err != nil {
		http.Error(w, "bad tty id", http.StatusBadRequest)
		return
	}
	s.mu.RLock()
	h := s.tty
	s.mu.RUnlock()
	if h == nil {
		http.Error(w, "no terminal handler", http.StatusServiceUnavailable)
		return
	}
	h(port, w, r)
}

// FilePath is the path under which the desktop app serves local files
// through the Wails asset handler (no network involved).
const FilePath = "/edex-file"

// ServeFile serves the local file given by the "path" query parameter, with
// Range support for audio and video playback. It replaces file:// URLs.
func ServeFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return
	}
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	static := s.static
	s.mu.RUnlock()
	if static == nil {
		http.NotFound(w, r)
		return
	}
	http.FileServer(http.FS(static)).ServeHTTP(w, r)
}

// Event is a message pushed to the UI, like webContents.send(channel, ...args).
type Event struct {
	Channel string `json:"channel"`
	Args    []any  `json:"args"`
}

type eventClient struct {
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &eventClient{conn: conn, send: make(chan []byte, 256)}
	s.clientsMu.Lock()
	s.clients[c] = struct{}{}
	s.clientsMu.Unlock()

	go func() {
		for msg := range c.send {
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
		conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	s.clientsMu.Lock()
	if _, ok := s.clients[c]; ok {
		delete(s.clients, c)
		close(c.send)
	}
	s.clientsMu.Unlock()
}

// Emit broadcasts an event to every connected UI.
func (s *Server) Emit(channel string, args ...any) {
	if args == nil {
		args = []any{}
	}
	msg, err := json.Marshal(Event{Channel: channel, Args: args})
	if err != nil {
		log.Printf("bridge: cannot encode event %s: %v", channel, err)
		return
	}
	s.clientsMu.Lock()
	defer s.clientsMu.Unlock()
	for c := range s.clients {
		select {
		case c.send <- msg:
		default:
			// Slow client: drop it rather than blocking the backend.
			delete(s.clients, c)
			close(c.send)
		}
	}
}

// URL returns the base URL of the server.
func (s *Server) URL() string { return fmt.Sprintf("http://127.0.0.1:%d", s.Port) }

// Arg decodes the i-th RPC argument into v. Missing arguments leave v
// untouched.
func Arg(args []json.RawMessage, i int, v any) error {
	if i >= len(args) {
		return nil
	}
	return json.Unmarshal(args[i], v)
}
