package network

import (
	"context"
	"encoding/json"
	"fmt"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// AnnounceMsg represents a GossipSub message announcing a new or updated file.
type AnnounceMsg struct {
	Type         string `json:"type"`          // "NEW_FILE" or "CODEBOOK_UPDATE"
	Filename     string `json:"filename"`      // Basename of the .crom file
	OriginalSize uint64 `json:"original_size"` // Size of the original uncompressed file
	ChunkCount   uint32 `json:"chunk_count"`   // Total chunks
	Sender       string `json:"sender"`        // Peer ID of the announcer
}

// GossipManager handles pubsub operations for the node.
type GossipManager struct {
	node   *CromNode
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	ctx    context.Context
	cancel context.CancelFunc
}

// setupGossipSub initializes the GossipSub router and subscribes to the codebook topic.
func (n *CromNode) setupGossipSub() error {
	ctx, cancel := context.WithCancel(n.ctx)

	// Create a new PubSub service using the GossipSub router
	ps, err := pubsub.NewGossipSub(ctx, n.Host)
	if err != nil {
		cancel()
		return fmt.Errorf("gossip: new gossipsub: %w", err)
	}
	n.PubSub = ps

	// The topic is scoped to the network partition (CodebookHash)
	topicName := fmt.Sprintf("crom/announce/%x", n.CodebookHash[:16])

	topic, err := ps.Join(topicName)
	if err != nil {
		cancel()
		return fmt.Errorf("gossip: join topic: %w", err)
	}

	sub, err := topic.Subscribe()
	if err != nil {
		cancel()
		return fmt.Errorf("gossip: subscribe topic: %w", err)
	}

	gm := &GossipManager{
		node:   n,
		topic:  topic,
		sub:    sub,
		ctx:    ctx,
		cancel: cancel,
	}

	go gm.readLoop()

	return nil
}

// readLoop continuously reads messages from the subscription.
func (gm *GossipManager) readLoop() {
	for {
		msg, err := gm.sub.Next(gm.ctx)
		if err != nil {
			return // Context canceled or subscription closed
		}

		// Ignore our own messages
		if msg.ReceivedFrom == gm.node.Host.ID() {
			continue
		}

		var announce AnnounceMsg
		if err := json.Unmarshal(msg.Data, &announce); err != nil {
			fmt.Printf("[Gossip] Mensagem invalida recebida de %s\n", msg.ReceivedFrom)
			continue
		}

		fmt.Printf("\n📢 [Rede] Anuncio Recebido: %s tem novo arquivo '%s' (%d chunks)\n",
			announce.Sender, announce.Filename, announce.ChunkCount)
	}
}

// AnnounceFile publishes a NEW_FILE message to the network.
func (n *CromNode) AnnounceFile(ctx context.Context, filename string, originalSize uint64, chunkCount uint32) error {
	if n.PubSub == nil {
		return fmt.Errorf("gossip: pubsub not initialized")
	}

	topicName := fmt.Sprintf("crom/announce/%x", n.CodebookHash[:16])
	topic, err := n.PubSub.Join(topicName)
	if err != nil {
		return err
	}

	msg := AnnounceMsg{
		Type:         "NEW_FILE",
		Filename:     filename,
		OriginalSize: originalSize,
		ChunkCount:   chunkCount,
		Sender:       n.Host.ID().String(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := topic.Publish(ctx, data); err != nil {
		return fmt.Errorf("gossip: publish failed: %w", err)
	}

	return nil
}
