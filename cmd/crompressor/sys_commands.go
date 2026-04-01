//go:build !wasm

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/MrJc01/crompressor/internal/autobrain"
	"github.com/MrJc01/crompressor/internal/metrics"
	"github.com/MrJc01/crompressor/internal/network"
	"github.com/MrJc01/crompressor/internal/vfs"
	cvfs "github.com/MrJc01/crompressor/pkg/cromlib/vfs"
	"github.com/MrJc01/crompressor/internal/cromfs"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

func addSystemCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(daemonCmd())
	rootCmd.AddCommand(sharedDaemonCmd())
	rootCmd.AddCommand(shareCmd())
	rootCmd.AddCommand(keysCmd())
	rootCmd.AddCommand(trustCmd())
	rootCmd.AddCommand(mountCmd())
	rootCmd.AddCommand(cromfsCmd())
	rootCmd.AddCommand(llmVfsCmd())
}
func daemonCmd() *cobra.Command {
	var codebookPath, dataDir, encKey string
	var listenPort int
	var bootstrapAddrs []string
	var allowHiveMind bool
	cmd := &cobra.Command{Use: "daemon", Short: "Inicia o nó P2P do CROM para sincronização na rede soberana", Long: `Inicia um nó libp2p que participa da rede CROM definida pelo Codebook informado.
O nó irá descobrir outros peers via mDNS (LAN) e DHT (WAN), e permitirá a sincronização
de arquivos .crom na pasta data-dir.`, RunE: func(cmd *cobra.Command, args []string) error {
		if codebookPath == "" {
			return fmt.Errorf("--codebook é obrigatório")
		}
		if dataDir == "" {
			dataDir = "./crom-data"
		}
		fmt.Println("╔═══════════════════════════════════════════╗")
		fmt.Println("║           CROM P2P NODE (Daemon)          ║")
		fmt.Println("╠═══════════════════════════════════════════╣")
		fmt.Printf("║  Codebook:  %-29s ║\n", codebookPath)
		fmt.Printf("║  Data Dir:  %-29s ║\n", dataDir)
		fmt.Printf("║  Port:      %-29d ║\n", listenPort)
		fmt.Println("╚═══════════════════════════════════════════╝")
		metrics.InitCromMetrics()
		node, err := network.NewCromNode(codebookPath, listenPort, dataDir, encKey)
		if err != nil {
			return err
		}
		defer node.Stop()
		_, err = network.LoadIdentity()
		if err == nil {
			fmt.Printf("✔ Identidade Carregada (Ed25519)\n")
		} else {
			fmt.Printf("⚠ Aviso: Operando anonimamente (Sem identidade trust). Use 'crompressor keys --gen'\n")
		}
		if allowHiveMind {
			fmt.Printf("✔ Hive Mind Ativada: Sandboxing e Quarentena para Brains ativados.\n")
		} else {
			fmt.Printf("🔒 Hive Mind Desativada: GossipSub ignorará payloads de Brains externos.\n")
		}
		syncProto := network.NewSyncProtocol(node)
		if err := node.SetupDHT(bootstrapAddrs); err != nil {
			fmt.Printf("Aviso: DHT falhou em iniciar: %v\n", err)
		}
		fmt.Println("✔ Nó P2P iniciado com sucesso.")
		fmt.Printf("  Peer ID: %s\n", node.PeerID().String())
		fmt.Println("  Endereços:")
		for _, addr := range node.Addrs() {
			fmt.Printf("    - %s\n", addr)
		}
		fmt.Println("\nAguardando conexões soberanas e requisições de sync. (Ctrl+C para sair)")
		go func() {
			time.Sleep(2 * time.Second)
			files, _ := filepath.Glob(filepath.Join(dataDir, "*.crom"))
			for _, f := range files {
				info, err := os.Stat(f)
				if err == nil {
					_ = node.AnnounceFile(context.Background(), filepath.Base(f), 0, uint32(info.Size()/1024/1024))
				}
			}
		}()
		http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			peers := node.Host.Network().Peers()
			peerCount := len(peers)
			peerID := node.Host.ID().String()
			fmt.Fprintf(w, `{"peers": %d, "peerID": "%s", "version": "v16"}`, peerCount, peerID)
		})
		http.HandleFunc("/share", func(w http.ResponseWriter, r *http.Request) {
			fileParam := r.URL.Query().Get("file")
			if fileParam == "" {
				http.Error(w, "file param missing", http.StatusBadRequest)
				return
			}
			info, err := os.Stat(fileParam)
			if err != nil {
				http.Error(w, "file not found", http.StatusNotFound)
				return
			}
			err = node.AnnounceFile(context.Background(), filepath.Base(fileParam), 0, uint32(info.Size()/1024/1024))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "{\"status\": \"shared\"}")
		})
		http.Handle("/metrics", promhttp.Handler())
		go func() {
			if err := http.ListenAndServe("127.0.0.1:9099", nil); err != nil {
				fmt.Printf("Aviso: RPC HTTP Server falhou: %v\n", err)
			}
		}()
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nEncerrando nó CROM (daemon)...")
		_ = syncProto
		return nil
	}}
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb)")
	cmd.Flags().StringVarP(&dataDir, "data-dir", "d", "", "Diretório de dados para os arquivos .crom (default: ./crom-data)")
	cmd.Flags().IntVarP(&listenPort, "port", "p", 4001, "Porta TCP/QUIC do libp2p")
	cmd.Flags().StringSliceVar(&bootstrapAddrs, "bootstrap", nil, "Endereços multiaddr para bootstrap DHT")
	cmd.Flags().StringVarP(&encKey, "encrypt", "e", "", "Chave AES para ler os pacotes. Requerida se os .crom estiverem encriptados.")
	cmd.Flags().BoolVar(&allowHiveMind, "allow-hive-mind", false, "Habilita Quarentena e Aceitação de Brains via P2P (Opt-in Hive Mind)")
	return cmd
}
func sharedDaemonCmd() *cobra.Command {
	var socketPath string
	cmd := &cobra.Command{Use: "shared-daemon", Short: "Inicia o Unified Service Daemon (UDS/IPC) do Cérebro para Multi-Apps", Long: `Garante que apenas UMA cópia em RAM do Dicionário (Brain) sirva centenas de aplicativos Android/Servidor usando Sockets Locais ultrarrápidos (OOM Defense Limit O(1)).`, RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("╔═══════════════════════════════════════════╗")
		fmt.Println("║     CROM UNIVERSAL DAEMON (IPC Core)      ║")
		fmt.Printf("║  Socket: %-32s ║\n", socketPath)
		fmt.Println("╚═══════════════════════════════════════════╝")
		b := autobrain.NewSharedBrain(socketPath)
		if err := b.Start(); err != nil {
			return err
		}
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("\nTerminando Daemon Local IPC...")
		b.Stop()
		return nil
	}}
	cmd.Flags().StringVar(&socketPath, "socket", "/tmp/crompressor.sock", "Caminho do Socket UNIX")
	return cmd
}
func keysCmd() *cobra.Command {
	var gen bool
	cmd := &cobra.Command{Use: "keys", Short: "Gerencia a identidade soberana Ed25519 (Keyring)", RunE: func(cmd *cobra.Command, args []string) error {
		if gen {
			return network.GenerateIdentity()
		}
		return fmt.Errorf("use --gen para gerar uma nova identidade P2P")
	}}
	cmd.Flags().BoolVar(&gen, "gen", false, "Gera um novo par de chaves Ed25519")
	return cmd
}
func trustCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "trust <peerID>", Short: "Adiciona um Peer ID à Web of Trust para receber Codebooks", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		peerID := args[0]
		err := network.TrustPeer(peerID)
		if err != nil {
			return err
		}
		fmt.Printf("✔ Peer '%s' validado como Confiável para a Colmeia.\n", peerID)
		return nil
	}}
	return cmd
}
func shareCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "share <file.crom>", Short: "Anuncia um arquivo .crom na rede soberana P2P", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:9099/share?file=%s", filePath))
		if err != nil {
			return fmt.Errorf("falha ao contactar daemon: %v (o daemon está rodando?)", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("daemon retornou erro: status %d", resp.StatusCode)
		}
		fmt.Printf("✔ Arquivo '%s' anunciado com sucesso no enxame P2P.\n", filePath)
		return nil
	}}
	return cmd
}
func mountCmd() *cobra.Command {
	var input, mountPoint, codebookPath, encryptionKey string
	var cacheMB int
	cmd := &cobra.Command{Use: "mount", Short: "Monta um arquivo .crom como um VFS (Virtual Filesystem)", Long: `Monta o arquivo compactado em um diretório vazio para leitura sob demanda (Lazy Loading) do conteúdo original.`, RunE: func(cmd *cobra.Command, args []string) error {
		if input == "" || mountPoint == "" || codebookPath == "" {
			return fmt.Errorf("flags --input, --mountpoint e --codebook são obrigatórias")
		}
		isCloud := len(input) > 7 && (input[:7] == "http://" || input[:8] == "https://")
		fmt.Println("╔═══════════════════════════════════════════╗")
		if isCloud {
			fmt.Println("║   CROM VFS (Cloud-Native Remote Mount)    ║")
		} else {
			fmt.Println("║       CROM VFS (Virtual Filesystem)       ║")
		}
		fmt.Println("╠═══════════════════════════════════════════╣")
		fmt.Printf("║  Input:    %-30s ║\n", input)
		fmt.Printf("║  Mount:    %-30s ║\n", mountPoint)
		fmt.Printf("║  Codebook: %-30s ║\n", codebookPath)
		if isCloud {
			fmt.Printf("║  Mode:     HTTP Range Requests (S3/CDN)   ║\n")
		}
		if encryptionKey != "" {
			fmt.Printf("║  Security: AES-256-GCM Enabled            ║\n")
		}
		fmt.Println("╚═══════════════════════════════════════════╝")
		info, err := os.Stat(mountPoint)
		if os.IsNotExist(err) {
			return fmt.Errorf("mountpoint %s doesn't exist", mountPoint)
		}
		if !info.IsDir() {
			return fmt.Errorf("mountpoint %s is not a directory", mountPoint)
		}
		if err := vfs.Mount(input, mountPoint, codebookPath, encryptionKey, cacheMB); err != nil {
			return fmt.Errorf("erro na montagem VFS: %v", err)
		}
		return nil
	}}
	cmd.Flags().StringVarP(&input, "input", "i", "", "Caminho do arquivo .crom (aceita URLs HTTP/HTTPS para montagem remota S3/CDN)")
	cmd.Flags().StringVarP(&mountPoint, "mountpoint", "m", "", "Diretório de montagem (deve ser vazio)")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook (.cromdb)")
	cmd.Flags().StringVar(&encryptionKey, "encrypt", "", "Chave/Senha para descriptografia")
	cmd.Flags().IntVar(&cacheMB, "cache", 64, "Hard Limit em Megabytes para o Motor RAM L1/L2 (O(1) VFS)")
	return cmd
}
func cromfsCmd() *cobra.Command {
	var mountPoint, outPool, codebookPath string
	cmd := &cobra.Command{Use: "cromfs", Short: "Monta o Daemon de Escrita FUSE (Global Deduplication)", Long: `Monta um sistema de arquivos onde todas as escritas são compresso-compiladas via CROM.`, RunE: func(cmd *cobra.Command, args []string) error {
		if mountPoint == "" || outPool == "" || codebookPath == "" {
			return fmt.Errorf("flags --mountpoint, --out-pool e --codebook obrigatórias")
		}
		os.MkdirAll(outPool, 0755)
		return cromfs.Mount(mountPoint, outPool, codebookPath)
	}}
	cmd.Flags().StringVarP(&mountPoint, "mountpoint", "m", "", "Diretório virtual /mnt/cromfs")
	cmd.Flags().StringVarP(&outPool, "out-pool", "o", "", "Onde salvar (.crom)")
	cmd.Flags().StringVarP(&codebookPath, "codebook", "c", "", "Caminho do Codebook base")
	return cmd
}
func llmVfsCmd() *cobra.Command {
	var mountPoint string
	cmd := &cobra.Command{Use: "llm-vfs", Short: "Monta o Daemon VFS puramente O(1) Paging para LLMs (Out-of-Core)", Long: `Monta uma abstração CROM-FS no diretório especificado para hospedar arquivos .gguf virtuais gigantes e paginar requisições em O(1) via dicionário JIT, bypassando limites da RAM física.`, RunE: func(cmd *cobra.Command, args []string) error {
		if mountPoint == "" {
			return fmt.Errorf("flag --mountpoint obrigatória")
		}
		fmt.Println("╔═══════════════════════════════════════════╗")
		fmt.Println("║     CROM LLM-VFS (Out-Of-Core Paging)     ║")
		fmt.Println("╠═══════════════════════════════════════════╣")
		fmt.Printf("║  Target Mount: %-26s ║\n", mountPoint)
		fmt.Printf("║  Mode:         %-26s ║\n", "JIT / Zero-Copy")
		fmt.Println("╚═══════════════════════════════════════════╝")
		return cvfs.MountServer(mountPoint)
	}}
	cmd.Flags().StringVarP(&mountPoint, "mountpoint", "m", "", "Diretório virtual host /mnt/crom_llm")
	return cmd
}
