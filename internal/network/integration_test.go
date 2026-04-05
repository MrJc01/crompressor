//go:build !wasm

package network

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/MrJc01/crompressor/internal/codebook"
	"github.com/MrJc01/crompressor/pkg/cromlib"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func TestTwoNodeSync(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 1. Setup Data Dirs
	dirA := t.TempDir()
	dirB := t.TempDir()

	// 2. Find/Generate Codebook
	codebookPath := "../../testdata/trained.cromdb"
	if _, err := os.Stat(codebookPath); os.IsNotExist(err) {
		t.Skip("Codebook not found. Run 'make gen-codebook' first.")
	}

	// 3. Create a synthetic file and pack it in Node A's directory
	originalFile := filepath.Join(dirA, "source.txt")
	testData := []byte("CROM P2P Integration Test - Hello World! " +
		"This proves that the SyncProtocol works end-to-end.")
	if err := os.WriteFile(originalFile, testData, 0644); err != nil {
		t.Fatal(err)
	}

	cromFileA := filepath.Join(dirA, "source.txt.crom")
	opts := cromlib.DefaultPackOptions()
	if _, err := cromlib.Pack(originalFile, cromFileA, codebookPath, opts); err != nil {
		t.Fatalf("Failed to pack file: %v", err)
	}

	// 4. Start Node A (Sender)
	nodeA, err := NewCromNode(codebookPath, 0, dirA, "") // port 0 = random
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
	nodeB, err := NewCromNode(codebookPath, 0, dirB, "")
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
	err = syncProtoB.RequestSync(ctx, nodeA.PeerID(), "source.txt.crom")
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 9. Verify the received file on Node B
	cromFileB := filepath.Join(dirB, "source.txt.crom")
	if _, err := os.Stat(cromFileB); os.IsNotExist(err) {
		t.Fatal("Node B did not save the synchronized file")
	}

	restoredFileB := filepath.Join(dirB, "restored.txt")
	if err := cromlib.Unpack(cromFileB, restoredFileB, codebookPath, cromlib.DefaultUnpackOptions()); err != nil {
		t.Fatalf("Failed to unpack synchronized file: %v", err)
	}

	restoredData, err := os.ReadFile(restoredFileB)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(testData, restoredData) {
		t.Fatalf("Data mismatch!\nExpected: %s\nGot:      %s", testData, restoredData)
	}

	// Also verify that codebook open doesn't panic on the new nodes
	cb, _ := codebook.Open(codebookPath)
	cb.Close()

	// Wait for background connection streams to flush before teardown 
	// to prevent auth EOF panics under the -race detector schedule
	time.Sleep(500 * time.Millisecond)

	fmt.Println("✔ Integration test passed successfully!")
}
