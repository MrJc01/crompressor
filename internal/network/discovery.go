//go:build !wasm

package network

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
)

// DefaultBootstrapPeers returns the default set of IPFS bootstrap peers.
// In a real private network, these would be dedicated bootstrap nodes for CROM.
var DefaultBootstrapPeers = []string{
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
	"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
}

// SetupDHT initializes the Kademlia Distributed Hash Table for WAN discovery.
func (n *CromNode) SetupDHT(bootstrapAddrs []string) error {
	var err error
	n.DHT, err = dht.New(n.ctx, n.Host, dht.Mode(dht.ModeServer))
	if err != nil {
		return fmt.Errorf("discovery: failed to create DHT: %w", err)
	}

	if err = n.DHT.Bootstrap(n.ctx); err != nil {
		return fmt.Errorf("discovery: failed to bootstrap DHT: %w", err)
	}

	if len(bootstrapAddrs) == 0 {
		bootstrapAddrs = DefaultBootstrapPeers
	}

	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapAddrs {
		ma, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid bootstrap multiaddr %s: %v\n", peerAddr, err)
			continue
		}

		peerinfo, _ := peer.AddrInfoFromP2pAddr(ma)
		if peerinfo == nil {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := n.Host.Connect(n.ctx, *peerinfo); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: bootstrap connect to %s failed: %v\n", peerinfo.ID, err)
			}
		}()
	}
	wg.Wait()

	// Wait for connected peers and setup routing discovery
	fmt.Println("[Descoberta] DHT conectado. Realizando Rendezvous na rede WAN...")

	routingDiscovery := routing.NewRoutingDiscovery(n.DHT)

	// Rendezvous string is the codebook hash
	rendezvousStr := fmt.Sprintf("crom-network-%x", n.CodebookHash[:16])
	
	// Advertise this node
	util.Advertise(n.ctx, routingDiscovery, rendezvousStr)

	// Discover others
	go func() {
		for {
			peers, err := routingDiscovery.FindPeers(n.ctx, rendezvousStr)
			if err != nil {
				time.Sleep(10 * time.Second)
				continue
			}

			// Handle peers concurrently while listening
			for p := range peers {
				if p.ID == n.Host.ID() || len(p.Addrs) == 0 {
					continue
				}

				// Only consider non-connected
				if n.Host.Network().Connectedness(p.ID) == network.Connected {
					continue
				}

				fmt.Printf("[Descoberta] Peer DHT encontrado: %s\n", p.ID.String())

				// Connect and Authenticate
				go func(pi peer.AddrInfo) {
					ctxCtx, cancel := context.WithTimeout(n.ctx, 15*time.Second)
					defer cancel()

					if err := n.Host.Connect(ctxCtx, pi); err != nil {
						return
					}

					if err := n.AuthenticatePeer(ctxCtx, pi.ID); err != nil {
						fmt.Printf("[Descoberta] Auth com DHT peer %s falhou: %v\n", pi.ID.String(), err)
						return
					}

					fmt.Printf("[Descoberta] Conexão Soberana estabelecida (WAN) com %s\n", pi.ID.String())
				}(p)
			}

			time.Sleep(30 * time.Second) // Poll DHT every 30s
		}
	}()

	return nil
}
