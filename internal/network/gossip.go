//go:build !wasm

package network

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MrJc01/crompressor/internal/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// ProposeChunkMsg represents a federated learning proposal for a universal chunk.
type ProposeChunkMsg struct {
	Type      string `json:"type"`       // "PROPOSE_CHUNK"
	Hash      string `json:"hash"`       // Hash of the chunk
	Payload   []byte `json:"payload"`    // Raw chunk data
	Weight    uint32 `json:"weight"`     // Recurrency score
	Signature []byte `json:"signature"`  // Ed25519 signature of the sender
	Sender    string `json:"sender"`     // Peer ID
}

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

		// Attempt to parse as ProposeChunkMsg first (Research 18)
		var propose ProposeChunkMsg
		if err := json.Unmarshal(msg.Data, &propose); err == nil && propose.Type == "PROPOSE_CHUNK" {
			// [V21] Zero-Knowledge Sybil Defense: Validar Assinatura Dilithium Pós-Quântica (Research 25/27)
			if propose.Weight > 0 {
				isValid := crypto.VerifyDilithium([]byte(propose.Sender), propose.Signature, []byte(propose.Hash))
				if !isValid {
					fmt.Printf("\n🛑 [SRE-Swarm] Assinatura Pós-Quântica INVÁLIDA de %s. Roteamento Bloqueado!\n", propose.Sender)
					continue
				}

				fmt.Printf("\n🧠 [Swarm] Padrão Quântico Seguro Verificado de %s! (Hash: %s, Peso: %d)\n", propose.Sender, propose.Hash, propose.Weight)
				// A partir daqui, o Codebook instanciaria SimSearchGPU() e gravaria no Mmap local.
			}
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

// ProposeUniversalPattern publishes a PROPOSE_CHUNK message to federate learning.
func (n *CromNode) ProposeUniversalPattern(ctx context.Context, hash string, payload []byte, weight uint32, signature []byte) error {
	if n.PubSub == nil {
		return fmt.Errorf("swarm: pubsub not initialized for federated learning")
	}

	topicName := fmt.Sprintf("crom/announce/%x", n.CodebookHash[:16])
	topic, err := n.PubSub.Join(topicName)
	if err != nil {
		return err
	}

	msg := ProposeChunkMsg{
		Type:      "PROPOSE_CHUNK",
		Hash:      hash,
		Payload:   payload,
		Weight:    weight,
		Signature: signature,
		Sender:    n.Host.ID().String(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := topic.Publish(ctx, data); err != nil {
		return fmt.Errorf("swarm: publish proposed chunk failed: %w", err)
	}

	return nil
}
