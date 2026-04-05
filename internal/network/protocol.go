//go:build !wasm

package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/MrJc01/crompressor/internal/codebook"
	cromsync "github.com/MrJc01/crompressor/pkg/sync"
)

const (
	// SyncProtocolID is the identifier for the CROM manifest exchange protocol.
	SyncProtocolID = "/crom/sync/1.0"
)

// Message Types
const (
	MsgSyncReq      byte = 0x01 // Request a manifest for a specific original file hash (or filename)
	MsgManifest     byte = 0x02 // The serialized ChunkManifest
	MsgDiffReq      byte = 0x03 // Request missing chunks (payload is array of indices)
	MsgChunkData    byte = 0x04 // Raw delta payload
	MsgCodebookHash byte = 0x05 // 32-byte SHA-256 BuildHash of the codebook
	MsgCodebookReq  byte = 0x06 // Request the full .cromdb binary
	MsgCodebookData byte = 0x07 // Response: full .cromdb binary
	MsgError        byte = 0xFF // Error message
)

// SyncProtocol handles manifest exchange and chunk transfers.
type SyncProtocol struct {
	node *CromNode
}

// NewSyncProtocol registers the sync stream handler.
func NewSyncProtocol(node *CromNode) *SyncProtocol {
	p := &SyncProtocol{node: node}
	node.Host.SetStreamHandler(SyncProtocolID, p.handleStream)
	return p
}

// handleStream processes incoming requests from other peers.
func (p *SyncProtocol) handleStream(s network.Stream) {
	defer s.Close()

	for {
		// Read msg type
		msgType := make([]byte, 1)
		if _, err := io.ReadFull(s, msgType); err != nil {
			if err != io.EOF {
				fmt.Printf("[Sync] Peer %s disconectou: %v\n", s.Conn().RemotePeer(), err)
			}
			return
		}

		// Read payload length
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(s, lenBuf); err != nil {
			return
		}
		payloadLen := binary.LittleEndian.Uint32(lenBuf)

		// Read payload
		payload := make([]byte, payloadLen)
		if _, err := io.ReadFull(s, payload); err != nil {
			return
		}

		switch msgType[0] {
		case MsgSyncReq:
			filename := string(payload)
			fmt.Printf("[Sync] Peer %s requisitou manifest de '%s'\n", s.Conn().RemotePeer(), filename)
			p.handleSyncReq(s, filename)

		case MsgDiffReq:
			fmt.Printf("[Sync] Peer %s solicitou chunks do arquivo\n", s.Conn().RemotePeer())
			p.handleDiffReq(s, payload)

		case MsgCodebookHash:
			fmt.Printf("[Sync] Peer %s enviou hash do codebook\n", s.Conn().RemotePeer())
			p.handleCodebookHash(s, payload)

		case MsgCodebookReq:
			fmt.Printf("[Sync] Peer %s requisitou codebook binário\n", s.Conn().RemotePeer())
			p.handleCodebookReq(s)

		default:
			fmt.Printf("[Sync] Mensagem desconhecida de %s: 0x%02x\n", s.Conn().RemotePeer(), msgType[0])
		}
	}
}

// handleSyncReq finds the local .crom file, generates its manifest, and sends it.
func (p *SyncProtocol) handleSyncReq(s network.Stream, filename string) {
	// Security: prevent path traversal
	cleanName := filepath.Base(filename)
	if !strings.HasSuffix(cleanName, ".crom") {
		cleanName += ".crom"
	}

	localPath := filepath.Join(p.node.DataDir, cleanName)

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		sendError(s, "Arquivo nao encontrado na seed")
		return
	}

	manifest, err := cromsync.GenerateManifest(localPath, p.node.CodebookPath, p.node.EncKey)
	if err != nil {
		sendError(s, "Erro ao gerar manifest: "+err.Error())
		return
	}

	bin := manifest.ToBinary()
	sendMsg(s, MsgManifest, bin)
}

// handleDiffReq handles a request for missing chunks.
// Payload format:
//
//	[Filename length (2 bytes)][Filename bytes][Number of indices(4 bytes LE)][Index array (uint32 LE)...]
func (p *SyncProtocol) handleDiffReq(s network.Stream, payload []byte) {
	if len(payload) < 6 {
		sendError(s, "Payload invalido")
		return
	}

	nameLen := binary.LittleEndian.Uint16(payload[0:2])
	if len(payload) < int(2+nameLen+4) {
		sendError(s, "Payload truncado")
		return
	}

	filename := string(payload[2 : 2+nameLen])
	cleanName := filepath.Base(filename)
	if !strings.HasSuffix(cleanName, ".crom") {
		cleanName += ".crom"
	}
	localPath := filepath.Join(p.node.DataDir, cleanName)

	offset := uint32(2 + nameLen)
	numIndices := binary.LittleEndian.Uint32(payload[offset : offset+4])
	offset += 4

	if uint32(len(payload)) < offset+numIndices*4 {
		sendError(s, "Lista de indices incompleta")
		return
	}

	indices := make([]uint32, numIndices)
	for i := uint32(0); i < numIndices; i++ {
		indices[i] = binary.LittleEndian.Uint32(payload[offset+i*4 : offset+i*4+4])
	}

	// Stream requested chunks
	err := StreamChunks(localPath, p.node.CodebookPath, p.node.EncKey, indices, s)
	if err != nil {
		fmt.Printf("[Sync] Erro no bitswap para %s: %v\n", s.Conn().RemotePeer(), err)
	}
}

// RequestSync is called proactively by a node to download a file from a remote peer.
// Flow:
// 0. Codebook Handshake (hash exchange, download if mismatch)
// 1. Sends SYNC_REQ
// 2. Receives MANIFEST
// 3. Diff against local (if file exists) or request all chunks
// 4. Send DIFF_REQ with missing indices
// 5. Receive CHUNK_DATA stream and rebuild
func (p *SyncProtocol) RequestSync(ctx context.Context, pid peer.ID, filename string) error {
	s, err := p.node.Host.NewStream(ctx, pid, SyncProtocolID)
	if err != nil {
		return fmt.Errorf("sync: open stream: %w", err)
	}
	defer s.Close()

	// 0. Codebook Handshake
	if err := sendMsg(s, MsgCodebookHash, p.node.CodebookHash[:]); err != nil {
		return fmt.Errorf("sync: enviar codebook hash: %w", err)
	}

	hashResp, hashPayload, err := readMsg(s)
	if err != nil {
		return fmt.Errorf("sync: ler resposta codebook hash: %w", err)
	}

	if hashResp == MsgError {
		return fmt.Errorf("sync: codebook handshake error: %s", string(hashPayload))
	}

	if hashResp == MsgCodebookHash {
		var remoteHash [32]byte
		copy(remoteHash[:], hashPayload)

		if !bytes.Equal(remoteHash[:], p.node.CodebookHash[:]) {
			fmt.Printf("[Sync] Codebook mismatch! Solicitando .cromdb do peer...\n")
			if err := sendMsg(s, MsgCodebookReq, nil); err != nil {
				return fmt.Errorf("sync: request codebook: %w", err)
			}

			cbResp, cbPayload, err := readMsg(s)
			if err != nil {
				return fmt.Errorf("sync: ler codebook data: %w", err)
			}
			if cbResp != MsgCodebookData {
				return fmt.Errorf("sync: esperava CODEBOOK_DATA (0x07), recebeu 0x%02x", cbResp)
			}

			remoteCbPath := filepath.Join(p.node.DataDir, "remote_peer.cromdb")
			if err := os.WriteFile(remoteCbPath, cbPayload, 0644); err != nil {
				return fmt.Errorf("sync: salvar codebook remoto: %w", err)
			}

			// Update node to use the remote codebook for this sync
			p.node.CodebookPath = remoteCbPath
			newCb, err := codebook.Open(remoteCbPath)
			if err == nil {
				p.node.CodebookHash = newCb.BuildHash()
				newCb.Close()
			}
			fmt.Printf("[Sync] ✔ Codebook remoto salvo em %s\n", remoteCbPath)
		} else {
			fmt.Printf("[Sync] ✔ Codebooks idênticos. Prosseguindo com sync.\n")
		}
	}

	// 1. Send Request
	if err := sendMsg(s, MsgSyncReq, []byte(filename)); err != nil {
		return err
	}

	// 2. Receive Manifest
	msgType, payload, err := readMsg(s)
	if err != nil {
		return fmt.Errorf("sync: read response: %w", err)
	}

	if msgType == MsgError {
		return fmt.Errorf("remote error: %s", string(payload))
	}

	if msgType != MsgManifest {
		return fmt.Errorf("esperava MANIFEST (0x02), recebeu 0x%02x", msgType)
	}

	remoteManifest, err := cromsync.FromBinary(payload)
	if err != nil {
		return fmt.Errorf("sync: parse remote manifest: %w", err)
	}

	fmt.Printf("[Sync] Recebido manifesto para '%s' (%d chunks totais)\n", filename, remoteManifest.ChunkCount)

	// 3. Compare with local (if any)
	destPath := filepath.Join(p.node.DataDir, filename)
	if !strings.HasSuffix(destPath, ".crom") {
		destPath += ".crom"
	}

	var missingIndices []uint32

	if _, err := os.Stat(destPath); err == nil {
		fmt.Printf("[Sync] Arquivo local encontrado. Analisando delta...\n")
		localManifest, err := cromsync.GenerateManifest(destPath, p.node.CodebookPath, p.node.EncKey)
		if err != nil {
			fmt.Printf("[Sync] Aviso: Erro ao ler manifesto local (%v). Baixando tudo.\n", err)
			for i := uint32(0); i < remoteManifest.ChunkCount; i++ {
				missingIndices = append(missingIndices, i)
			}
		} else {
			diffRes := cromsync.Diff(localManifest, remoteManifest)
			if len(diffRes.Missing) == 0 {
				fmt.Printf("[Sync] ✔ Arquivo local já está atualizado (0 chunks faltando).\n")
				return nil
			}

			type chunkKey struct{ CodebookID, DeltaHash uint64 }
			missingSet := make(map[chunkKey]struct{}, len(diffRes.Missing))
			for _, e := range diffRes.Missing {
				missingSet[chunkKey{e.CodebookID, e.DeltaHash}] = struct{}{}
			}

			for i, e := range remoteManifest.Entries {
				if _, ok := missingSet[chunkKey{e.CodebookID, e.DeltaHash}]; ok {
					missingIndices = append(missingIndices, uint32(i))
				}
			}
			fmt.Printf("[Sync] Diferença lógica detectada: %d chunks faltando.\n", len(missingIndices))
		}
	} else {
		missingIndices = make([]uint32, remoteManifest.ChunkCount)
		for i := uint32(0); i < remoteManifest.ChunkCount; i++ {
			missingIndices[i] = i
		}
	}

	if len(missingIndices) == 0 {
		return nil
	}

	// 4. Send Diff Request
	diffPayload := make([]byte, 2+len(filename)+4+len(missingIndices)*4)
	binary.LittleEndian.PutUint16(diffPayload[0:2], uint16(len(filename)))
	copy(diffPayload[2:], filename)

	offset := 2 + len(filename)
	binary.LittleEndian.PutUint32(diffPayload[offset:], uint32(len(missingIndices)))
	offset += 4

	for i, idx := range missingIndices {
		binary.LittleEndian.PutUint32(diffPayload[offset+i*4:], idx)
	}

	if err := sendMsg(s, MsgDiffReq, diffPayload); err != nil {
		return fmt.Errorf("sync: enviar diff req: %w", err)
	}

	// 5. Receive Chunks and Rebuild .crom
	fmt.Printf("[Sync] Iniciando bitswap reverso de %d chunks faltantes...\n", len(missingIndices))
	tempPath := destPath + ".tmp"
	err = ReceiveChunks(tempPath, destPath, remoteManifest, missingIndices, s, p.node.CodebookPath, p.node.EncKey)
	if err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("sync: bitswap merge error: %w", err)
	}

	// Replace old file with new merged file
	os.Remove(destPath)
	os.Rename(tempPath, destPath)

	fmt.Printf("[Sync] ✔ Sincronismo Delta P2P de '%s' finalizado com sucesso.\n", filename)
	return nil
}

// --- Codebook Sharing Handlers ---

// handleCodebookHash responds with our own codebook hash for comparison.
func (p *SyncProtocol) handleCodebookHash(s network.Stream, payload []byte) {
	// Reply with our own hash so the requester can compare
	sendMsg(s, MsgCodebookHash, p.node.CodebookHash[:])
}

// handleCodebookReq sends the full .cromdb binary to the requesting peer.
func (p *SyncProtocol) handleCodebookReq(s network.Stream) {
	data, err := os.ReadFile(p.node.CodebookPath)
	if err != nil {
		sendError(s, "Erro ao ler codebook: "+err.Error())
		return
	}
	fmt.Printf("[Sync] Enviando codebook binário (%d bytes)\n", len(data))
	sendMsg(s, MsgCodebookData, data)
}

// --- Wire Format Helpers ---

func sendMsg(s network.Stream, msgType byte, payload []byte) error {
	header := make([]byte, 5)
	header[0] = msgType
	binary.LittleEndian.PutUint32(header[1:5], uint32(len(payload)))

	if _, err := s.Write(header); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := s.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

func sendError(s network.Stream, errMsg string) {
	sendMsg(s, MsgError, []byte(errMsg))
}

func readMsg(s network.Stream) (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(s, header); err != nil {
		return 0, nil, err
	}

	msgType := header[0]
	length := binary.LittleEndian.Uint32(header[1:5])

	if length > 100*1024*1024 { // Sanity check: 100MB max per message
		return 0, nil, fmt.Errorf("payload too large: %d bytes", length)
	}

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(s, payload); err != nil {
			return 0, nil, err
		}
	}

	return msgType, payload, nil
}
