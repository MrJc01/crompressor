package autobrain

import (
	"fmt"
	"net"
	"os"
	"sync"
)

// SharedBrain represents the singleton UNIX Daemon serving the Reality Compiler
// to multiple apps in the same SO at identical Codebook Cost O(1).
type SharedBrain struct {
	SocketPath string
	listener   net.Listener
	mu         sync.Mutex
	stopChan   chan struct{}
}

// NewSharedBrain initializes the Multi-app Unified Service (V21).
func NewSharedBrain(socketPath string) *SharedBrain {
	if socketPath == "" {
		socketPath = "/tmp/crompressor.sock"
	}
	return &SharedBrain{
		SocketPath: socketPath,
		stopChan:   make(chan struct{}),
	}
}

// Start opens the UNIX IPC domain socket allowing inter-process binary messaging.
func (b *SharedBrain) Start() error {
	// Remover socket inativo pendente (OOM Defense / Anti-Lock)
	if err := os.RemoveAll(b.SocketPath); err != nil {
		return err
	}

	ln, err := net.Listen("unix", b.SocketPath)
	if err != nil {
		return err
	}
	b.listener = ln
	fmt.Printf("🧠 [IPC-Daemon] Codebook Singleton Compartilhado ouvindo em: %s\n", b.SocketPath)

	go func() {
		for {
			conn, err := b.listener.Accept()
			if err != nil {
				select {
				case <-b.stopChan:
					return
				default:
					fmt.Printf("shared_daemon SRE erro non-fatal: %v\n", err)
					continue
				}
			}
			go b.handleConnection(conn)
		}
	}()
	return nil
}

// handleConnection intercepts raw bytes from App A, B or C and returns universal Pointers from the shared memory.
func (b *SharedBrain) handleConnection(conn net.Conn) {
	defer conn.Close()
	// Mock Base of the API Protocol (Production gRPC or Custom Binary Handshake)
	conn.Write([]byte("ACK_CROM_DAEMON_V21\n"))
}

// Stop cleanly detaches and unbinds the local OS Unix Socket file.
func (b *SharedBrain) Stop() {
	close(b.stopChan)
	if b.listener != nil {
		b.listener.Close()
	}
	os.RemoveAll(b.SocketPath)
}
