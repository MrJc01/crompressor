package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/MrJc01/crompressor/pkg/sdk"
	"github.com/gorilla/websocket"
	"github.com/zserge/lorca"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type FileEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	IsDir    bool   `json:"is_dir"`
	IsCrom   bool   `json:"is_crom"`
	IsCromdb bool   `json:"is_cromdb"`
}

func jsonResp(w http.ResponseWriter, code int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}

var (
	startTime = time.Now()
	bus       *sdk.EventBus
	comp      sdk.Compressor
	vault     sdk.Vault
	swarm     sdk.Swarm
	cryptoSvc sdk.Crypto
	identity  sdk.Identity
)

func main() {
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║   CROMPRESSOR v0.1.0 (Full Engine)       ║")
	fmt.Println("╠═══════════════════════════════════════════╣")
	fmt.Println("║ SDK: Compressor+Vault+Swarm+Crypto+ID    ║")
	fmt.Println("╚═══════════════════════════════════════════╝")

	bus = sdk.NewEventBus()
	comp = sdk.NewCompressor(bus)
	vault = sdk.NewVault(bus)
	swarm = sdk.NewSwarm(bus)
	cryptoSvc = sdk.NewCrypto()
	identity = sdk.NewIdentity()

	// Start sovereignty watchdog
	vault.StartWatchdog(context.Background())

	// ─── FILE MANAGEMENT ───
	http.HandleFunc("/api/list", handleList)
	http.HandleFunc("/api/info", handleInfo)
	http.HandleFunc("/api/delete", handleDelete)
	http.HandleFunc("/api/health", handleHealth)
	http.HandleFunc("/api/open", handleOpen)
	http.HandleFunc("/api/mkdir", handleMkdir)
	http.HandleFunc("/api/mkfile", handleMkfile)

	// ─── CROM ENGINE ───
	http.HandleFunc("/api/pack", handlePack)
	http.HandleFunc("/api/unpack", handleUnpack)
	http.HandleFunc("/api/train", handleTrain)
	http.HandleFunc("/api/verify", handleVerify)

	// ─── VFS ───
	http.HandleFunc("/api/mount", handleMount)
	http.HandleFunc("/api/unmount", handleUnmount)

	// ─── CRYPTO ───
	http.HandleFunc("/api/encrypt", handleEncrypt)
	http.HandleFunc("/api/decrypt", handleDecrypt)

	// ─── P2P SWARM ───
	http.HandleFunc("/api/swarm/start", handleSwarmStart)
	http.HandleFunc("/api/swarm/stop", handleSwarmStop)
	http.HandleFunc("/api/swarm/peers", handleSwarmPeers)

	// ─── IDENTITY ───
	http.HandleFunc("/api/identity/generate", handleIdentityGenerate)

	// ─── WEBSOCKET ───
	http.HandleFunc("/ws", handleWebSocket)

	// ─── STATIC FILES ───
	http.Handle("/", http.FileServer(http.Dir("ui/dist")))

	port := "9100"
	url := fmt.Sprintf("http://localhost:%s", port)
	fmt.Printf("✔ Crompressor rodando em %s\n", url)
	fmt.Println("  17 endpoints ativos: list, info, delete, health, pack, unpack, train, verify, mount, unmount, encrypt, decrypt, swarm/*, identity/*")
	
	go func() {
		if err := http.ListenAndServe("127.0.0.1:"+port, nil); err != nil {
			log.Fatal("HTTPServer erro:", err)
		}
	}()

	ui, err := lorca.New(url, "", 1100, 800)
	if err != nil {
		fmt.Println("Erro ao iniciar interface nativa, fallback para browser...")
		openBrowser(url)
		select {} // Block indefinitely
	}
	defer ui.Close()

	// Mantém o go rodando até fecharem a janela do App
	<-ui.Done()
}

// ─── HANDLERS ───

func handleList(w http.ResponseWriter, r *http.Request) {
	dir := r.URL.Query().Get("dir")
	if dir == "" {
		dir, _ = os.UserHomeDir()
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		jsonResp(w, 400, APIResponse{Error: err.Error()})
		return
	}
	files := []FileEntry{}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") { continue }
		info, err := e.Info()
		if err != nil { continue }
		files = append(files, FileEntry{
			Name: e.Name(), Path: filepath.Join(dir, e.Name()), Size: info.Size(),
			Modified: info.ModTime().Format("02/01/2006 15:04"), IsDir: e.IsDir(),
			IsCrom: strings.HasSuffix(e.Name(), ".crom"), IsCromdb: strings.HasSuffix(e.Name(), ".cromdb"),
		})
	}
	jsonResp(w, 200, APIResponse{Success: true, Data: map[string]interface{}{"dir": dir, "files": files, "count": len(files)}})
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" { jsonResp(w, 400, APIResponse{Error: "path required"}); return }
	info, err := os.Stat(path)
	if err != nil { jsonResp(w, 404, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Data: map[string]interface{}{
		"name": info.Name(), "size": info.Size(), "modified": info.ModTime().Format("02/01/2006 15:04:05"),
		"is_dir": info.IsDir(), "is_crom": strings.HasSuffix(info.Name(), ".crom"),
	}})
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string `json:"path"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Path == "" { jsonResp(w, 400, APIResponse{Error: "path required"}); return }
	if err := os.Remove(req.Path); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: "Removed: " + req.Path})
}

func handleOpen(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string `json:"path"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Path == "" { jsonResp(w, 400, APIResponse{Error: "path required"}); return }
	go openBrowser(req.Path)
	jsonResp(w, 200, APIResponse{Success: true, Message: "Opened natively: " + req.Path})
}

func handleMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string `json:"path"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Path == "" { jsonResp(w, 400, APIResponse{Error: "path required"}); return }
	if err := os.MkdirAll(req.Path, 0755); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: "Mkdir: " + req.Path})
}

func handleMkfile(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string `json:"path"` }
	if json.NewDecoder(r.Body).Decode(&req) != nil || req.Path == "" { jsonResp(w, 400, APIResponse{Error: "path required"}); return }
	if err := os.WriteFile(req.Path, []byte(""), 0644); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: "Mkfile: " + req.Path})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResp(w, 200, APIResponse{Success: true, Data: map[string]interface{}{
		"status": "running", "version": "0.1.0", "uptime": time.Since(startTime).String(),
		"modules": []string{"compressor", "vault", "swarm", "crypto", "identity"},
	}})
}

func handlePack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input    string `json:"input"`
		Output   string `json:"output"`
		Codebook string `json:"codebook"`
		Key      string `json:"key"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	if req.Input == "" || req.Output == "" || req.Codebook == "" { jsonResp(w, 400, APIResponse{Error: "input, output, codebook required"}); return }
	go comp.Pack(context.Background(), sdk.PackCommand{InputPath: req.Input, OutputPath: req.Output, CodebookPath: req.Codebook, EncryptionKey: req.Key, Concurrency: 4})
	jsonResp(w, 200, APIResponse{Success: true, Message: "Pack started: " + req.Input})
}

func handleUnpack(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input    string  `json:"input"`
		Output   string  `json:"output"`
		Codebook string  `json:"codebook"`
		Key      string  `json:"key"`
		Fuzz     float64 `json:"fuzziness"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	go comp.Unpack(context.Background(), sdk.UnpackCommand{InputPath: req.Input, OutputPath: req.Output, CodebookPath: req.Codebook, EncryptionKey: req.Key, Fuzziness: req.Fuzz})
	jsonResp(w, 200, APIResponse{Success: true, Message: "Unpack started: " + req.Input})
}

func handleTrain(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input  string `json:"input"`
		Output string `json:"output"`
		Size   int    `json:"size"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	if req.Size == 0 { req.Size = 8192 }
	go comp.Train(context.Background(), sdk.TrainCommand{InputDir: req.Input, OutputPath: req.Output, MaxCodewords: req.Size})
	jsonResp(w, 200, APIResponse{Success: true, Message: "Training started"})
}

func handleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Original string `json:"original"`
		Restored string `json:"restored"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	match, err := comp.Verify(context.Background(), req.Original, req.Restored)
	if err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Data: map[string]interface{}{"match": match}})
}

func handleMount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CromFile string `json:"crom_file"`
		Mount    string `json:"mount"`
		Codebook string `json:"codebook"`
		Key      string `json:"key"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	go vault.Mount(context.Background(), req.CromFile, sdk.VaultOptions{MountPoint: req.Mount, CodebookPath: req.Codebook, EncryptionKey: req.Key})
	jsonResp(w, 200, APIResponse{Success: true, Message: "Mounting VFS: " + req.Mount})
}

func handleUnmount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mount string `json:"mount"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	if err := vault.Unmount(context.Background(), req.Mount); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: "Unmounted: " + req.Mount})
}

func handleEncrypt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input      string `json:"input"`
		Output     string `json:"output"`
		Passphrase string `json:"passphrase"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	if err := cryptoSvc.EncryptFile(req.Input, req.Output, req.Passphrase); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: "Encrypted: " + req.Output})
}

func handleDecrypt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input      string `json:"input"`
		Output     string `json:"output"`
		Passphrase string `json:"passphrase"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	if err := cryptoSvc.DecryptFile(req.Input, req.Output, req.Passphrase); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: "Decrypted: " + req.Output})
}

func handleSwarmStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port    int    `json:"port"`
		DataDir string `json:"data_dir"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil { jsonResp(w, 400, APIResponse{Error: "invalid JSON"}); return }
	if req.Port == 0 { req.Port = 4001 }
	if err := swarm.Start(context.Background(), req.Port, req.DataDir); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: fmt.Sprintf("Swarm started on port %d", req.Port)})
}

func handleSwarmStop(w http.ResponseWriter, r *http.Request) {
	if err := swarm.Stop(); err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Message: "Swarm stopped"})
}

func handleSwarmPeers(w http.ResponseWriter, r *http.Request) {
	peers, err := swarm.GetPeers()
	if err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Data: peers})
}

func handleIdentityGenerate(w http.ResponseWriter, r *http.Request) {
	pub, _, err := identity.GenerateKeypair()
	if err != nil { jsonResp(w, 500, APIResponse{Error: err.Error()}); return }
	jsonResp(w, 200, APIResponse{Success: true, Data: map[string]interface{}{"public_key": fmt.Sprintf("%x", pub)}})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil { return }
	defer conn.Close()

	allTypes := []sdk.EventType{
		sdk.EventVFSMounted, sdk.EventVFSUnmounted, sdk.EventPeerJoined, sdk.EventPeerLeft,
		sdk.EventSyncProg, sdk.EventAlertKill,
		"pack_progress", "pack_done", "pack_error",
		"unpack_done", "unpack_error",
		"train_progress", "train_done", "train_error",
		"verify_done",
	}
	channels := make([]<-chan sdk.SystemEvent, len(allTypes))
	for i, t := range allTypes {
		channels[i] = bus.Subscribe(t)
	}

	for {
		for _, ch := range channels {
			select {
			case ev := <-ch:
				bs, _ := json.Marshal(ev)
				_ = conn.WriteMessage(websocket.TextMessage, bs)
			default:
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":  err = exec.Command("xdg-open", url).Start()
	case "windows": err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin": err = exec.Command("open", url).Start()
	}
	if err != nil { fmt.Printf("Acesse %s manualmente.\n", url) }
}
