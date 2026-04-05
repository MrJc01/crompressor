package sdk

import (
	"context"
	"fmt"

	"github.com/MrJc01/crompressor/internal/network"
)

type DefaultSwarm struct {
	eventBus *EventBus
	node     *network.CromNode
}

func NewSwarm(bus *EventBus) Swarm {
	return &DefaultSwarm{
		eventBus: bus,
	}
}

func (s *DefaultSwarm) Start(ctx context.Context, port int, dataDir string) error {
	// Codebook is required for node instantiation. We mock passing an empty or generic one for now.
	// In the real implementation the codebook validates the sovereign ring.
	cbPath := "default.cromdb"
	encKey := ""

	node, err := network.NewCromNode(cbPath, port, dataDir, encKey)
	if err != nil {
		return err
	}
	s.node = node
	
	_ = network.NewSyncProtocol(node)

	// Emit GUI alert
	if s.eventBus != nil {
		s.eventBus.Emit(EventPeerJoined, node.PeerID().String())
	}
	return nil
}

func (s *DefaultSwarm) Stop() error {
	if s.node != nil {
		s.node.Stop()
		if s.eventBus != nil {
			s.eventBus.Emit(EventPeerLeft, s.node.PeerID().String())
		}
		s.node = nil
	}
	return nil
}

func (s *DefaultSwarm) GetPeers() ([]PeerInfo, error) {
	if s.node == nil {
		return nil, fmt.Errorf("swarm not started")
	}

	rawPeers := s.node.Host.Network().Peers()
	peers := make([]PeerInfo, 0, len(rawPeers))
	
	for _, p := range rawPeers {
		peers = append(peers, PeerInfo{
			ID: p.String(),
			// In production, we extract multiaddr
		})
	}
	return peers, nil
}

func (s *DefaultSwarm) AnnounceManifest(manifest []byte) error {
	if s.node == nil {
		return fmt.Errorf("swarm not started")
	}
	return s.node.AnnounceFile(context.Background(), "sdk-announced.crom", 0, uint32(len(manifest)))
}

func (s *DefaultSwarm) WatchActiveFolder(ctx context.Context, folderPath string) error {
	if s.eventBus != nil {
		s.eventBus.Emit(EventSyncProg, map[string]interface{}{
			"folder": folderPath,
			"status": "started",
		})
	}
	// Simulated active sync / polling for now to provide UI feedback
	go func() {
		// Here would be fsnotify.Watcher logic
		<-ctx.Done()
		if s.eventBus != nil {
			s.eventBus.Emit(EventSyncProg, map[string]interface{}{
				"folder": folderPath,
				"status": "stopped",
			})
		}
	}()
	return nil
}
