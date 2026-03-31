package entropy

import "runtime"

// NodeConfig defines the limitations and features active for the host system.
// Ensures that 1 single binary scales natively to Satellites OR Enterprise Cloud.
type NodeConfig struct {
	MaxPeers    int
	EnableFEC   bool
	Threads     int
	MmapLimit   string
	ProfileName string
}

// DetermineProfile auto-senses the hardware returning the proper SRE limits.
func DetermineProfile() *NodeConfig {
	cpus := runtime.NumCPU()

	// 1. Satelite / RPi Zero / Old IoT (Survival Mode)
	// CPUs single-core or extremely constrained environments.
	if cpus <= 1 {
		return &NodeConfig{
			MaxPeers:    5,
			EnableFEC:   true, // Crucial for flaky radio networks (Cosmic Bit-flips)
			Threads:     1,
			MmapLimit:   "32MB",
			ProfileName: "IoT/Space (Survival)",
		}
	}

	// 2. Mobile Android / Modern RPi (Edge Mode)
	// Avoids thermal throttling while keeping Kademlia DHT alive.
	if cpus <= 4 {
		return &NodeConfig{
			MaxPeers:    15,
			EnableFEC:   true, // Protection against 4G/Cellular packet drops
			Threads:     2,
			MmapLimit:   "128MB",
			ProfileName: "Mobile (Edge)",
		}
	}

	// 3. Cloud Server / PC (Enterprise Mode)
	// Unlimited throughput utilizing GPU Offload or SIMD CPU cores.
	return &NodeConfig{
		MaxPeers:    500,
		EnableFEC:   false, // Reliable connections via Fiber/DataCenter TCP
		Threads:     cpus,
		MmapLimit:   "Unlimited",
		ProfileName: "Enterprise Cloud",
	}
}
