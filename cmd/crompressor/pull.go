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
	var listenPort, timeoutSec int

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Baixa de um peer apenas o que falta para montar um arquivo .crom",
		RunE: func(cmd *cobra.Command, args []string) error {
			if peerAddr == "" || file == "" || codebookPath == "" {
				return fmt.Errorf("flags --peer, --file e --codebook são obrigatórias")
			}
			ma, err := multiaddr.NewMultiaddr(peerAddr)
			if err != nil {
				return fmt.Errorf("endereço do peer inválido: %w", err)
			}
			info, err := peer.AddrInfoFromP2pAddr(ma)
			if err != nil {
				return fmt.Errorf("endereço do peer precisa terminar em /p2p/<id>: %w", err)
			}

			node, err := network.NewCromNode(codebookPath, listenPort, dataDir, encKey)
			if err != nil {
				return err
			}
			defer node.Stop()
			syncProto := network.NewSyncProtocol(node)

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
			defer cancel()
			if err := node.Host.Connect(ctx, *info); err != nil {
				return fmt.Errorf("falha ao conectar: %w", err)
			}
			if err := node.AuthenticatePeer(ctx, info.ID); err != nil {
				return fmt.Errorf("falha na autenticação: %w", err)
			}
			if err := syncProto.RequestSync(ctx, info.ID, file); err != nil {
				return err
			}
			fmt.Printf("✔ Recebidos %d bytes pela rede.\n", syncProto.BytesReceived)
			return nil
		},
	}

	cmd.Flags().StringVar(&peerAddr, "peer", "", "Multiaddr do peer, terminando em /p2p/<id>")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Nome do arquivo .crom no peer")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb)")
	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", "./crom-data", "Diretório local dos arquivos .crom")
	cmd.Flags().StringVarP(&encKey, "encrypt", "e", "", "Chave AES dos arquivos locais")
	cmd.Flags().IntVarP(&listenPort, "port", "p", 0, "Porta local (0 = aleatória)")
	cmd.Flags().IntVar(&timeoutSec, "timeout", 600, "Tempo limite em segundos")
	return cmd
}
