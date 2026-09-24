package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type WSInfo struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

type WebSocketServer struct {
	manager *Manager
	Token   string
	Port    int
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewWebSocketServer(manager *Manager) (*WebSocketServer, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(bytes)

	return &WebSocketServer{
		manager: manager,
		Token:   token,
	}, nil
}

func (s *WebSocketServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/", s.handleWS)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	s.Port = listener.Addr().(*net.TCPAddr).Port

	go func() {
		if err := http.Serve(listener, mux); err != nil {
			log.Printf("WebSocket server error: %v", err)
		}
	}()

	return nil
}

func (s *WebSocketServer) handleWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token != s.Token {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/ws/")
	if id == "" {
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	session, err := s.manager.GetSession(id)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	// Read from PTY -> Send to WS
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := session.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from PTY: %v", err)
				}
				return
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}()

	// Read from WS -> Send to PTY
	for {
		msgType, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
			session.Write(p)
		}
	}
}
