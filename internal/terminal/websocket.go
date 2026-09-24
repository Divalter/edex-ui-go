package terminal

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	// Only used by the development server, which checks its token before
	// upgrading.
	CheckOrigin: func(*http.Request) bool { return true },
}

type wsSink struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (s *wsSink) Send(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return s.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (s *wsSink) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second))
	s.conn.Close()
}

// ServeWS attaches a WebSocket client to the terminal (development server).
func (t *TTY) ServeWS(w http.ResponseWriter, r *http.Request) {
	if t.closed.Load() {
		http.Error(w, "tty closed", http.StatusGone)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	sink := &wsSink{conn: conn}
	t.Attach(sink)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil || t.Write(msg) != nil {
			break
		}
	}
	if t.Detach(sink) && !t.main {
		t.Close()
	}
}
