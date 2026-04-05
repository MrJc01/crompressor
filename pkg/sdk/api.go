package sdk

import (
	"context"
)

// ProgressEvent is emitted during long-running tasks like Pack/Unpack.
type ProgressEvent struct {
	Phase      string  `json:"phase"`       // e.g "hashing", "chunking", "packing", "completed", "error: ..."
	Percentage float64 `json:"percentage"`  // 0.0 to 100.0
	BytesRead  uint64  `json:"bytes_read"`
	TotalBytes uint64  `json:"total_bytes"`
}

// PackCommand arguments for compiling a file.
type PackCommand struct {
	InputPath     string
	OutputPath    string
	CodebookPath  string
	EncryptionKey string
	Concurrency   int
}

// UnpackCommand arguments for restoring a file.
type UnpackCommand struct {
	InputPath     string
	OutputPath    string
	CodebookPath  string
	EncryptionKey string
	Fuzziness     float64
}

// Compressor defines the core CROM Engine capabilities.
type Compressor interface {
	Pack(ctx context.Context, cmd PackCommand) (<-chan ProgressEvent, error)
	Unpack(ctx context.Context, cmd UnpackCommand) (<-chan ProgressEvent, error)
	Train(ctx context.Context, cmd TrainCommand) (<-chan ProgressEvent, error)
	Verify(ctx context.Context, originalPath, restoredPath string) (bool, error)
}

// VaultOptions for VFS behavior.
type VaultOptions struct {
	AllowOther      bool
	MountPoint      string
	CodebookPath    string
	EncryptionKey   string
	KillSwitchEvent chan<- string
	WriteBackCache  string
}

// Vault manages VFS (FUSE) sovereign drives.
type Vault interface {
	Mount(ctx context.Context, cromFile string, opts VaultOptions) error
	Unmount(ctx context.Context, mountPoint string) error
	StartWatchdog(ctx context.Context) error
}

// PeerInfo represents a connected node in the Swarm.
type PeerInfo struct {
	ID        string   `json:"id"`
	Multiaddr []string `json:"multiaddr"`
	LatencyMs int64    `json:"latency_ms"`
}

// Swarm controls the P2P synchronization.
type Swarm interface {
	Start(ctx context.Context, port int, dataDir string) error
	Stop() error
	GetPeers() ([]PeerInfo, error)
	AnnounceManifest(manifest []byte) error
	WatchActiveFolder(ctx context.Context, folderPath string) error
}

// Identity manages Sovereign Ed25519 Keys.
type Identity interface {
	GenerateKeypair() (public, private []byte, err error)
	LoadKeypair(path string) error
	Sign(data []byte) ([]byte, error)
	Verify(public, data, signature []byte) bool
}

// Note: Crypto interface is defined in crypto.go
