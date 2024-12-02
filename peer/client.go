package peer

import (
	"context"
	"rabbit-ws/connection"
	"rabbit-ws/connection_pool"
	"rabbit-ws/tunnel"
	"rabbit-ws/tunnel_pool"
	"math/rand"
)

type ClientPeer struct {
	Peer
}

func NewClientPeer(tunnelNum int, endpoints []string, cipher tunnel.Cipher) ClientPeer {
	if initRand() != nil {
		panic("Error when initialize random seed.")
	}
	peerID := rand.Uint32()
	return newClientPeerWithID(peerID, tunnelNum, endpoints, cipher)
}

func newClientPeerWithID(peerID uint32, tunnelNum int, endpoints []string, cipher tunnel.Cipher) ClientPeer {
	peerCtx, removePeerFunc := context.WithCancel(context.Background())

	poolManager := tunnel_pool.NewClientManager(tunnelNum, endpoints, peerID, cipher)
	tunnelPool := tunnel_pool.NewTunnelPool(peerID, &poolManager, peerCtx)
	connectionPool := connection_pool.NewConnectionPool(tunnelPool, false, peerCtx)

	return ClientPeer{
		Peer: Peer{
			peerID:         peerID,
			connectionPool: *connectionPool,
			tunnelPool:     *tunnelPool,
			ctx:            peerCtx,
			cancel:         removePeerFunc,
		},
	}
}

func (cp *ClientPeer) Dial(address string) connection.Connection {
	conn := cp.connectionPool.NewPooledInboundConnection()
	conn.SendConnect(address)
	return conn
}
