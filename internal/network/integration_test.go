//go:build !wasm

package network

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/internal/trainer"
	"github.com/MrJc01/crompressor/pkg/cromlib"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func TestTwoNodeSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Run("plain", func(t *testing.T) { runTwoNodeSync(t, "") })
	t.Run("encrypted", func(t *testing.T) { runTwoNodeSync(t, "segredo") })
}

func runTwoNodeSync(t *testing.T, key string) {
	t.Setenv("CROM_TEST_DISABLE_MDNS", "1")

	// 1. Setup Data Dirs
	dirA := t.TempDir()
	dirB := t.TempDir()

	// 2. Both nodes share a brain trained on the old version of the file
	oldData := make([]byte, 1024*1024)
	rand.New(rand.NewSource(11)).Read(oldData)

	trainDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(trainDir, "old.bin"), oldData, 0644); err != nil {
		t.Fatal(err)
	}
	codebookPath := filepath.Join(t.TempDir(), "shared.cromdb")
	trainOpts := trainer.DefaultTrainOptions()
	trainOpts.InputDir = trainDir
	trainOpts.OutputPath = codebookPath
	trainOpts.ChunkSize = 4096
	trainOpts.MaxCodewords = 1024
	if _, err := trainer.Train(trainOpts); err != nil {
		t.Fatal(err)
	}

	// 3. Node A has the new version, with one byte inserted; node B has the old one
	testData := append([]byte("X"), oldData...)
	pack := func(data []byte, dir string) {
		t.Helper()
		src := filepath.Join(dir, "source.bin")
		if err := os.WriteFile(src, data, 0644); err != nil {
			t.Fatal(err)
		}
		packOpts := cromlib.DefaultPackOptions()
		packOpts.EncryptionKey = key
		if _, err := cromlib.Pack(src, filepath.Join(dir, "source.bin.crom"), codebookPath, packOpts); err != nil {
			t.Fatalf("Failed to pack file: %v", err)
		}
	}
	pack(testData, dirA)
	pack(oldData, dirB)

	// 4. Start Node A (Sender)
	nodeA, err := NewCromNode(codebookPath, 0, dirA, key) // port 0 = random
	if err != nil {
		t.Fatalf("Failed to start Node A: %v", err)
	}
	defer nodeA.Stop()

	syncProtoA := NewSyncProtocol(nodeA)
	_ = syncProtoA

	// Get Node A's address for B to connect directly without discovery
	addrsA := nodeA.Host.Addrs()
	if len(addrsA) == 0 {
		t.Fatal("Node A has no listen addresses")
	}
	fullAddrA := fmt.Sprintf("%s/p2p/%s", addrsA[0].String(), nodeA.PeerID().String())
	maA, err := multiaddr.NewMultiaddr(fullAddrA)
	if err != nil {
		t.Fatal(err)
	}
	peerInfoA, _ := peer.AddrInfoFromP2pAddr(maA)

	// 5. Start Node B (Receiver)
	nodeB, err := NewCromNode(codebookPath, 0, dirB, key)
	if err != nil {
		t.Fatalf("Failed to start Node B: %v", err)
	}
	defer nodeB.Stop()

	syncProtoB := NewSyncProtocol(nodeB)

	// 6. Connect Node B to Node A
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := nodeB.Host.Connect(ctx, *peerInfoA); err != nil {
		t.Fatalf("Node B failed to connect to A: %v", err)
	}

	// Allow protocol handlers to fully register after connection
	time.Sleep(500 * time.Millisecond)

	// 7. Sovereignty Handshake (retry up to 3 times to handle mDNS race)
	var authErr error
	for attempt := 0; attempt < 3; attempt++ {
		authErr = nodeB.AuthenticatePeer(ctx, nodeA.PeerID())
		if authErr == nil {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if authErr != nil {
		t.Fatalf("Authentication failed after retries: %v", authErr)
	}

	// 8. Node B requests Sync
	fmt.Println("--- Inciando Sincronização ---")
	err = syncProtoB.RequestSync(ctx, nodeA.PeerID(), "source.bin.crom")
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 9. Verify the received file on Node B
	if limit := int64(len(testData)) / 10; syncProtoB.BytesReceived > limit {
		t.Errorf("sync read %d bytes from the network, want at most %d", syncProtoB.BytesReceived, limit)
	}

	cromFileB := filepath.Join(dirB, "source.bin.crom")
	if _, err := os.Stat(cromFileB); os.IsNotExist(err) {
		t.Fatal("Node B did not save the synchronized file")
	}

	restoredFileB := filepath.Join(dirB, "restored.bin")
	if key != "" {
		raw, err := os.ReadFile(cromFileB)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(raw, testData[5000:5064]) || !bytes.Contains(raw[:8], []byte("CROM")) {
			t.Error("synced file should be an encrypted .crom")
		}
	}
	unpackOpts := cromlib.DefaultUnpackOptions()
	unpackOpts.EncryptionKey = key
	if err := cromlib.Unpack(cromFileB, restoredFileB, codebookPath, unpackOpts); err != nil {
		t.Fatalf("Failed to unpack synchronized file: %v", err)
	}

	restoredData, err := os.ReadFile(restoredFileB)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(testData, restoredData) {
		t.Fatal("Data mismatch after sync")
	}

	// Also verify that codebook open doesn't panic on the new nodes
	cb, _ := codebook.Open(codebookPath)
	cb.Close()

	// Wait for background connection streams to flush before teardown 
	// to prevent auth EOF panics under the -race detector schedule
	time.Sleep(500 * time.Millisecond)

	fmt.Println("✔ Integration test passed successfully!")
}
