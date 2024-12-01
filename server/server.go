package server

import (
	"github.com/ihciah/rabbit-tcp/logger"
	"github.com/ihciah/rabbit-tcp/peer"
	"github.com/ihciah/rabbit-tcp/tunnel"
	"net"
	"sync"
	"rabbit-tcp-MTCP-ws/wsconn"
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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}

	// Wrap WebSocket connection
	conn := websocketconn.NewWebSocketConn(wsConn)

	// Pass conn to your application logic
	handleConnection(conn)
}

func ServeThread(address string, ss *Server, wg *sync.WaitGroup) error {
	defer wg.Done()
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
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

func wsServeThread(address string, ss *Server, wg *sync.WaitGroup) error {
	defer wg.Done()
	//listener, err := net.Listen("tcp", address)
	wsListener := wsconn.NewWebSocketListener(address)
	
	go func() {
		log.Printf("WebSocket server listening on %s\n", address)
		err := http.ListenAndServe(address, wsListener)
		if err != nil {
			log.Fatalf("HTTP server error: %v\n", err)
		}
	}()
	for {
		conn, err := wsListener.Accept()
		if err != nil {
			log.Println("Listener closed:", err)
			continue
		}

		err = ss.peerGroup.AddTunnelFromConn(conn)
		if err != nil {
			ss.logger.Errorf("Error when add tunnel to tunnel pool: %v.\n", err)
		}
	}
	
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
		wg.Add(1)
		go ServeThread(address,s, &wg)
	}
	wg.Wait()
	return nil
}

func (s *Server) wsServe(addresses []string) error {
	var wg sync.WaitGroup
	for _,address:= range addresses {
		wg.Add(1)
		go wsServeThread(address,s, &wg)
	}
	wg.Wait()
	return nil
}
