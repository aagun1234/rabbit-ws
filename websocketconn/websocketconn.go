package websocketconn

import (
    "fmt"
    "net"
	"net/http"
	"sync"
	"errors"
    "time"
	"strings"
    "github.com/gorilla/websocket"
)

type websocketConn struct {
    conn *websocket.Conn
}

type WebSocketListener struct {
	upgrader websocket.Upgrader
	conns    chan net.Conn
	addr     net.Addr
	mu       sync.Mutex
	closed   bool
}

// NewWebSocketConn 是创建 websocketConn 的工厂函数
func NewwebsocketConn(conn *websocket.Conn) *websocketConn {
    return &websocketConn{conn: conn}
}

// Read 实现了 net.Conn 接口中的 Read 方法
func (wc *websocketConn) Read(b []byte) (n int, err error) {
    _, msg, err := wc.conn.ReadMessage()
    if err != nil {
        return 0, err
    }
    copy(b, msg)
    return len(msg), nil
}

// Write 实现了 net.Conn 接口中的 Write 方法
func (wc *websocketConn) Write(b []byte) (n int, err error) {
    err = wc.conn.WriteMessage(websocket.TextMessage, b)
    if err != nil {
        return 0, err
    }
    return len(b), nil
}

// Close 实现了 net.Conn 接口中的 Close 方法
func (wc *websocketConn) Close() error {
    return wc.conn.Close()
}

// LocalAddr 实现了 net.Conn 接口中的 LocalAddr 方法
func (wc *websocketConn) LocalAddr() net.Addr {
    return nil
}

// RemoteAddr 实现了 net.Conn 接口中的 RemoteAddr 方法
func (wc *websocketConn) RemoteAddr() net.Addr {
    return nil
}

// SetDeadline 实现了 net.Conn 接口中的 SetDeadline 方法
func (wc *websocketConn) SetDeadline(t time.Time) error {
    return nil
}

// SetReadDeadline 实现了 net.Conn 接口中的 SetReadDeadline 方法
func (wc *websocketConn) SetReadDeadline(t time.Time) error {
    return nil
}

// SetWriteDeadline 实现了 net.Conn 接口中的 SetWriteDeadline 方法
func (wc *websocketConn) SetWriteDeadline(t time.Time) error {
    return nil
}

// ConnectToServer 根据地址判断连接类型，并返回相应的 net.Conn 实现
func ConnectToServer(network, address string) (net.Conn, error) {
    if strings.HasPrefix(address, "ws://") || strings.HasPrefix(address, "wss://") {
        // 如果是 WebSocket 地址，建立 WebSocket 连接
        url := address
        if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
            url = "ws://" + url
        }
        conn, _, err := websocket.DefaultDialer.Dial(url, nil)
        if err != nil {
            return nil, fmt.Errorf("failed to connect WebSocket: %w", err)
        }
        return NewwebsocketConn(conn), nil
    } else {
        // 如果是普通的 TCP 地址，建立 TCP 连接
        conn, err := net.Dial(network, address)
        if err != nil {
            return nil, fmt.Errorf("failed to connect TCP: %w", err)
        }
        return conn, nil
    }
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

	conn := NewwebsocketConn(wsConn)
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
		wsConn := NewwebsocketConn(conn)
		handler(wsConn)
	})

	// 启动 HTTP 服务来监听 WebSocket 连接
	return http.ListenAndServe(address, nil)
}

