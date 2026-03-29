package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MrJc01/crompressor/pkg/sdk"
)

func main() {
	// Initialize the Native Crompressor SDK Wrapper (nil for no EventBus)
	packager := sdk.NewCompressor(nil)

	// Configure SRE-ready settings
	config := sdk.PackCommand{
		InputPath:    "data.json",
		OutputPath:   "output.crom",
		CodebookPath: "brain.cromdb",
		Concurrency:  4,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Println("🚀 Compressing API Logs using Native SDK...")

	// Launch Compression!
	progressChan, err := packager.Pack(ctx, config)
	if err != nil {
		log.Fatalf("Pack failed to start: %v", err)
	}

	// Consume the progress channel
	for p := range progressChan {
		if p.Phase == "completed" {
			fmt.Printf("✅ Success! Total Packed Bytes: %d\n", p.TotalBytes)
		} else if p.Phase == "packing" {
			fmt.Printf("... Packed %d bytes\n", p.BytesRead)
		} else {
			fmt.Printf("... %s\n", p.Phase)
		}
	}
}
