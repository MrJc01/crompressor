//go:build !wasm

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"

	"github.com/MrJc01/crompressor/internal/network"
)

func pullCmd() *cobra.Command {
	var peerAddr, file, codebookPath, dataDir, encKey string
	var bootstrapAddrs []string
	var privateDHT bool
	var listenPort, timeoutSec int

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Baixa de um peer apenas o que falta para montar um arquivo .crom",
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" || codebookPath == "" {
				return fmt.Errorf("flags --file e --codebook são obrigatórias")
			}
			if peerAddr == "" && len(bootstrapAddrs) == 0 {
				return fmt.Errorf("informe --peer ou --bootstrap")
			}

			node, err := network.NewCromNode(codebookPath, listenPort, dataDir, encKey)
			if err != nil {
				return err
			}
			defer node.Stop()
			syncProto := network.NewSyncProtocol(node)

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
			defer cancel()

			if peerAddr != "" {
				ma, err := multiaddr.NewMultiaddr(peerAddr)
				if err != nil {
					return fmt.Errorf("endereço do peer inválido: %w", err)
				}
				info, err := peer.AddrInfoFromP2pAddr(ma)
				if err != nil {
					return fmt.Errorf("endereço do peer precisa terminar em /p2p/<id>: %w", err)
				}
				if err := node.Host.Connect(ctx, *info); err != nil {
					return fmt.Errorf("falha ao conectar: %w", err)
				}
				if err := authenticate(ctx, node, info.ID); err != nil {
					return fmt.Errorf("falha na autenticação: %w", err)
				}
				if err := syncProto.RequestSync(ctx, info.ID, file); err != nil {
					return err
				}
			} else {
				if err := node.SetupDHT(bootstrapAddrs, privateDHT); err != nil {
					return err
				}
				if err := pullFromAnyPeer(ctx, node, syncProto, file); err != nil {
					return err
				}
			}
			fmt.Printf("✔ Recebidos %d bytes pela rede.\n", syncProto.BytesReceived)
			return nil
		},
	}

	cmd.Flags().StringVar(&peerAddr, "peer", "", "Multiaddr do peer, terminando em /p2p/<id>")
	cmd.Flags().StringSliceVar(&bootstrapAddrs, "bootstrap", nil, "Sem --peer: procura pela DHT a partir destes endereços")
	cmd.Flags().BoolVar(&privateDHT, "private-dht", false, "Usa uma DHT própria da rede CROM, em vez da DHT pública")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Nome do arquivo .crom no peer")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb)")
	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", "./crom-data", "Diretório local dos arquivos .crom")
	cmd.Flags().StringVarP(&encKey, "encrypt", "e", "", "Chave AES dos arquivos locais")
	cmd.Flags().IntVarP(&listenPort, "port", "p", 0, "Porta local (0 = aleatória)")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 600, "Tempo limite em segundos")
	return cmd
}

// authenticate retries because mDNS may start a handshake with the same peer at the same moment.
func authenticate(ctx context.Context, node *network.CromNode, id peer.ID) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = node.AuthenticatePeer(ctx, id); err == nil {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return err
}

// pullFromAnyPeer tries the connected peers until one has the file.
func pullFromAnyPeer(ctx context.Context, node *network.CromNode, syncProto *network.SyncProtocol, file string) error {
	var lastErr error
	for ctx.Err() == nil {
		for _, id := range node.Host.Network().Peers() {
			if err := authenticate(ctx, node, id); err != nil {
				lastErr = err
				continue
			}
			if lastErr = syncProto.RequestSync(ctx, id, file); lastErr == nil {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("nenhum peer entregou '%s' a tempo (último erro: %v)", file, lastErr)
}
