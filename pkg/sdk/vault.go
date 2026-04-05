package sdk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/MrJc01/crompressor/internal/vfs"
)

type DefaultVault struct {
	eventBus      *EventBus
	codebookPath  string
	watchdogStop  chan struct{}
}

func NewVault(bus *EventBus) Vault {
	return &DefaultVault{
		eventBus:     bus,
		watchdogStop: make(chan struct{}),
	}
}

func (v *DefaultVault) Mount(ctx context.Context, cromFile string, opts VaultOptions) error {
	v.codebookPath = opts.CodebookPath

	if v.eventBus != nil {
		v.eventBus.Emit(EventVFSMounted, map[string]string{
			"file":  cromFile,
			"mount": opts.MountPoint,
		})
	}

	// Mount blocks — run in goroutine for GUI responsiveness
	go func() {
		err := vfs.Mount(cromFile, opts.MountPoint, opts.CodebookPath, opts.EncryptionKey, 256)
		if v.eventBus != nil {
			v.eventBus.Emit(EventVFSUnmounted, map[string]interface{}{
				"file":  cromFile,
				"error": fmt.Sprintf("%v", err),
			})
		}
	}()

	// Auto-start watchdog for this mount
	go v.watchCodebook(opts.CodebookPath, opts.MountPoint)

	return nil
}

// Unmount detaches the FUSE volume via fusermount.
func (v *DefaultVault) Unmount(ctx context.Context, mountPoint string) error {
	cmd := exec.CommandContext(ctx, "fusermount", "-u", mountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmount failed: %s: %w", string(output), err)
	}

	if v.eventBus != nil {
		v.eventBus.Emit(EventVFSUnmounted, map[string]string{
			"mount": mountPoint,
		})
	}
	return nil
}

// StartWatchdog monitors OS signals AND codebook file presence.
func (v *DefaultVault) StartWatchdog(ctx context.Context) error {
	// Signal-based kill switch
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			if v.eventBus != nil {
				v.eventBus.Emit(EventAlertKill, "Kill-Switch: OS signal received")
			}
		case <-ctx.Done():
		}
	}()

	return nil
}

// watchCodebook polls for the codebook file every 2 seconds.
// If it disappears, emits sovereignty_kill and force-unmounts.
func (v *DefaultVault) watchCodebook(codebookPath, mountPoint string) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(codebookPath); os.IsNotExist(err) {
				// SOVEREIGNTY BREACH — Codebook removed!
				if v.eventBus != nil {
					v.eventBus.Emit(EventAlertKill, map[string]string{
						"reason":   "Codebook file removed from disk",
						"codebook": codebookPath,
						"mount":    mountPoint,
					})
				}

				// Force unmount
				exec.Command("fusermount", "-u", mountPoint).Run()
				return
			}
		case <-v.watchdogStop:
			return
		}
	}
}
