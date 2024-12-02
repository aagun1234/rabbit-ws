package server

import (
	"rabbit-ws/logger"
	"rabbit-ws/peer"
	"rabbit-ws/tunnel"
	"net"
	"net/http"
	"sync"
	"strings"
	"github.com/gorilla/websocket"
	"rabbit-ws/websocketconn"
)
type Server struct {
	peerGroup peer.PeerGroup
	logger    *logger.Logger
}

func NewServer(cipher tunnel.Cipher) Server {
	return Server{
		peerGroup: peer.NewPeerGroup(cipher),
		logger:    logger.NewLogger("[Server]"),
	}
}

// var upgrader = websocket.Upgrader{
    // CheckOrigin: func(r *http.Request) bool {
        // return true
    // },
// }
// func handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
    ////升级 HTTP 请求为 WebSocket 连接
    // conn, err := upgrader.Upgrade(w, r, nil)
    // if err != nil {
        // log.Printf("Failed to upgrade to WebSocket: %v", err)
        // return
    // }
    ////将 WebSocket 连接封装为 net.Conn 接口
    // wsConn := websocketconn.NewWebSocketConn(conn)
    
    ////这里可以继续进行处理，像处理普通的 net.Conn 一样
    // go handleConnection(wsConn)
// }

func wsServeThread(address string, ss *Server, wg *sync.WaitGroup) error {
	defer wg.Done()
	
	url := address
	idx := strings.Index(address, "s://")
    if idx != -1 {
        url = address[idx+len("s://"):]
    }
	idx =strings.Index(url, "/")
	listenaddr := url
	uri:="/"
	if idx != -1 {
        listenaddr = url[:idx]
        uri = url[idx+1:]
	} else {
	    listenaddr= url
		uri="/"
	}
	

	// 创建 WebSocket 监听器（HTTP server）
	httpListener, err := net.Listen("tcp", listenaddr) // 监听 WebSocket 的端口
	if err != nil {
		return err
	}
	ss.logger.Infoln("Listening for WebSocket connections on "+listenaddr)
	defer httpListener.Close()

	
	// WebSocket 升级及处理
	// 通过 http.Server 来处理 WebSocket 请求
	httpServer := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// WebSocket 升级逻辑
			ss.logger.Infoln("Upgrading websocket for "+uri)
			upgrader := websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool {
					// 检查请求路径是否为 /websocket
					return r.URL.Path == uri
				},
			}

			// 升级为 WebSocket 连接
			wsConnUpgraded, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				ss.logger.Errorf("Error upgrading WebSocket connection: %v.\n", err)
				return
			}

			// 将 WebSocket 连接封装为 net.Conn 类型
			wc := websocketconn.NewwebsocketConn(wsConnUpgraded)

			// 将 WebSocket 连接交给 peerGroup 处理
			err = ss.peerGroup.AddTunnelFromConn(wc)
			if err != nil {
				ss.logger.Errorf("Error adding WebSocket tunnel: %v.\n", err)
			}

		}),
	}

	// 启动 HTTP 服务器
	err = httpServer.Serve(httpListener)
	if err != nil {
		ss.logger.Errorf("Error starting HTTP server: %v.\n", err)
	}

	// 阻塞等待结束
	select {}
}

func ServeThread(address string, ss *Server, wg *sync.WaitGroup) error {
	defer wg.Done()
	
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	ss.logger.Infoln("Listening for TCP connections on "+address)
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			ss.logger.Errorf("Error when accept connection: %v.\n", err)
			continue
		}
		err = ss.peerGroup.AddTunnelFromConn(conn)
		if err != nil {
			ss.logger.Errorf("Error when add tunnel to tunnel pool: %v.\n", err)
		}
	}
}

func (s *Server) Serve(addresses []string) error {
        var wg sync.WaitGroup
	for _,address:= range addresses {
		if strings.HasPrefix(address, "ws://") || strings.HasPrefix(address, "wss://") {
			wg.Add(1)
			go wsServeThread(address,s, &wg)
		} else {
			wg.Add(1)
			go ServeThread(address,s, &wg)
		}
	}
	wg.Wait()
	return nil
}
