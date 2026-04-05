package vfs

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fuse"
)

// SovereigntyWatcher monitors critical system conditions and triggers automatic
// unmount of the FUSE filesystem when sovereignty is compromised.
//
// Triggers:
//  1. Codebook file is deleted or becomes inaccessible (polling every 1s)
//  2. OS signals (SIGINT, SIGTERM) for graceful shutdown
//  3. Manual stop via the stopCh channel (e.g., key invalidation)
type SovereigntyWatcher struct {
	server       *fuse.Server
	codebookPath string
	mountPoint   string
	stopCh       chan struct{}
}

// NewSovereigntyWatcher creates a new watcher bound to a FUSE server instance.
func NewSovereigntyWatcher(server *fuse.Server, codebookPath string, mountPoint string) *SovereigntyWatcher {
	return &SovereigntyWatcher{
		server:       server,
		codebookPath: codebookPath,
		mountPoint:   mountPoint,
		stopCh:       make(chan struct{}),
	}
}

// Start begins monitoring in the background. It returns immediately.
// The watcher will unmount and print a reason when triggered.
func (w *SovereigntyWatcher) Start() {
	// Signal handler
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Codebook polling ticker
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		defer ticker.Stop()
		defer signal.Stop(sigCh)

		for {
			select {
			case sig := <-sigCh:
				fmt.Fprintf(os.Stderr, "\n⚡ Sinal recebido (%v). Desmontando VFS...\n", sig)
				w.unmount()
				return

			case <-ticker.C:
				if _, err := os.Stat(w.codebookPath); os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "\n🛡️ SOBERANIA VIOLADA: Codebook removido (%s). Desmontagem forçada!\n", w.codebookPath)
					w.unmount()
					return
				}

			case <-w.stopCh:
				fmt.Fprintf(os.Stderr, "\n🔒 Chave invalidada. Desmontagem forçada!\n")
				w.unmount()
				return
			}
		}
	}()
}

// Stop triggers manual unmount (e.g., when encryption key is wiped from memory).
func (w *SovereigntyWatcher) Stop() {
	select {
	case w.stopCh <- struct{}{}:
	default:
	}
}

func (w *SovereigntyWatcher) unmount() {
	if err := w.server.Unmount(); err != nil {
		fmt.Fprintf(os.Stderr, "vfs: erro ao desmontar: %v\n", err)
	} else {
		fmt.Fprintf(os.Stderr, "✔ VFS desmontado com sucesso: %s\n", w.mountPoint)
	}
}
