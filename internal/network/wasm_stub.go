//go:build wasm

package network

func IsPeerTrusted(peerID string) bool { return false }
