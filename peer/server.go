package peer

import (
	"context"
	"rabbit-ws/connection_pool"
	"rabbit-ws/tunnel_pool"
)

type ServerPeer struct {
	Peer
}

func NewServerPeerWithID(peerID uint32, peerContext context.Context, removePeerFunc context.CancelFunc) ServerPeer {
	poolManager := tunnel_pool.NewServerManager(removePeerFunc)
	tunnelPool := tunnel_pool.NewTunnelPool(peerID, &poolManager, peerContext)

	connectionPool := connection_pool.NewConnectionPool(tunnelPool, true, peerContext)

	return ServerPeer{
		Peer: Peer{
			peerID:         peerID,
			connectionPool: *connectionPool,
			tunnelPool:     *tunnelPool,
			ctx:            peerContext,
			cancel:         removePeerFunc,
		},
	}
}
