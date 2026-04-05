//go:build wasm

package main

import (
	"github.com/spf13/cobra"
)

func addSystemCommands(rootCmd *cobra.Command) {
	// Comandos dependentes do SO (Daemon, FUSE, P2P) omitidos no WASM
}
