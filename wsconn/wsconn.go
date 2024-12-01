package wsconn

import (
	"fmt"
	"net/http"
	"github.com/gorilla/websocket"
	"time"
	"net"
	"sync"
	"errors"
)

// WebSocketConn 是我们实现的 net.Conn 接口的适配器，它封装了 websocket.Conn
type WebSocketConn struct {
	*websocket.Conn
}

// NewWebSocketConn 创建一个 WebSocketConn 实例
func NewWebSocketConn(conn *websocket.Conn) *WebSocketConn {
	return &WebSocketConn{Conn: conn}
}

// Read 方法实现 net.Conn 接口中的 Read
func (ws *WebSocketConn) Read(b []byte) (n int, err error) {
	_, msg, err := ws.Conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	copy(b, msg)
	return len(msg), nil
}

// Write 方法实现 net.Conn 接口中的 Write
func (ws *WebSocketConn) Write(b []byte) (n int, err error) {
	err = ws.Conn.WriteMessage(websocket.TextMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close 方法实现 net.Conn 接口中的 Close
func (ws *WebSocketConn) Close() error {
	return ws.Conn.Close()
}

// LocalAddr 方法实现 net.Conn 接口中的 LocalAddr
func (ws *WebSocketConn) LocalAddr() net.Addr {
	// WebSocket 不直接提供 LocalAddr，我们返回一个假设的地址
	// 可以根据实际情况替换为实际的 IP 地址或类似的信息
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

// RemoteAddr 方法实现 net.Conn 接口中的 RemoteAddr
func (ws *WebSocketConn) RemoteAddr() net.Addr {
	// WebSocket 不直接提供 RemoteAddr，我们返回一个假设的地址
	// 可以根据实际情况替换为实际的 IP 地址或类似的信息
	return &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 8080}
}

// SetDeadline 方法实现 net.Conn 接口中的 SetDeadline
func (ws *WebSocketConn) SetDeadline(t time.Time) error {
	err := ws.Conn.SetReadDeadline(t)
	if err != nil {
		return err
	}
	return ws.Conn.SetWriteDeadline(t)
}

// SetReadDeadline 方法实现 net.Conn 接口中的 SetReadDeadline
func (ws *WebSocketConn) SetReadDeadline(t time.Time) error {
	return ws.Conn.SetReadDeadline(t)
}

// SetWriteDeadline 方法实现 net.Conn 接口中的 SetWriteDeadline
func (ws *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return ws.Conn.SetWriteDeadline(t)
}

func WebSocketDial(network, address string) (net.Conn, error) {
	// WebSocket 地址需要加上 "ws://" 或 "wss://"
	url := "ws://" + address

	// 创建 WebSocket 连接
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	// 返回一个封装了 WebSocket 连接的 WebSocketConn 实例
	return NewWebSocketConn(conn), nil
}


type WebSocketListener struct {
	upgrader websocket.Upgrader
	conns    chan net.Conn
	addr     net.Addr
	mu       sync.Mutex
	closed   bool
}

// NewWebSocketListener creates a new WebSocketListener.
func NewWebSocketListener(addr string) *WebSocketListener {
	return &WebSocketListener{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins
			},
		},
		conns: make(chan net.Conn),
		addr:  &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 8080},
	}
}

// ServeHTTP handles HTTP upgrade requests to WebSocket.
func (l *WebSocketListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		http.Error(w, "WebSocket listener closed", http.StatusServiceUnavailable)
		return
	}

	wsConn, err := l.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}

	conn := NewWebSocketConn(wsConn)
	l.conns <- conn
}

// Accept waits for and returns the next connection.
func (l *WebSocketListener) Accept() (net.Conn, error) {
	conn, ok := <-l.conns
	if !ok {
		return nil, errors.New("listener closed")
	}
	return conn, nil
}

// Close stops the listener from accepting connections.
func (l *WebSocketListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return errors.New("listener already closed")
	}

	close(l.conns)
	l.closed = true
	return nil
}

// Addr returns the listener's network address.
func (l *WebSocketListener) Addr() net.Addr {
	return l.addr
}



func WebSocketListenAndServe(address string, handler func(conn net.Conn)) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("Error upgrading connection:", err)
			return
		}
		// 将 WebSocket 连接封装为 net.Conn 并传递给处理函数
		wsConn := NewWebSocketConn(conn)
		handler(wsConn)
	})

	// 启动 HTTP 服务来监听 WebSocket 连接
	return http.ListenAndServe(address, nil)
}

