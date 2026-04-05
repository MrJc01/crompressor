package vfs

import (
	"context"
	"fmt"
	"log"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// MountServer monta o VFS FUSE do Crompressor no caminho especificado.
func MountServer(mountpoint string) error {
	err := os.MkdirAll(mountpoint, 0755)
	if err != nil {
		return fmt.Errorf("falha ao criar ponto de montagem %s: %w", mountpoint, err)
	}

	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("cromvfs"),
		fuse.Subtype("cromfs"),
		fuse.ReadOnly(),
	)
	if err != nil {
		return fmt.Errorf("falha no mnt fuse: %w", err)
	}
	defer c.Close()

	log.Printf("[VFS-FUSE] Servidor Crompressor iniciado: %s", mountpoint)
	server := fs.New(c, &fs.Config{})

	err = server.Serve(FS{})
	if err != nil {
		return err
	}

	return nil
}

// FS implementa o VFS raiz
type FS struct{}

func (f FS) Root() (fs.Node, error) {
	return Dir{}, nil
}

// Dir implementa a pasta raiz virtual
type Dir struct{}

func (Dir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = 1
	a.Mode = os.ModeDir | 0555
	return nil
}

func (Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	// Expondo fisicamente arquivos ilusórios que a API do llama.cpp vai tentar acessar.
	if name == "deepseek-v3.gguf" || name == "llama-70b.gguf" {
		inode := uint64(2)
		if name == "llama-70b.gguf" {
			inode = 3
		}
		return File{Name: name, Inode: inode}, nil
	}
	return nil, fuse.ENOENT
}

func (Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	return []fuse.Dirent{
		{Inode: 2, Name: "deepseek-v3.gguf", Type: fuse.DT_File},
		{Inode: 3, Name: "llama-70b.gguf", Type: fuse.DT_File},
	}, nil
}

// File implementa o arquivo virtual GGUF
type File struct {
	Name  string
	Inode uint64
}

func (f File) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = f.Inode
	a.Mode = 0444
	// Simula 70 GB (75.161.927.680 bytes) virtuais exatos
	a.Size = 75161927680
	return nil
}

func (f File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// A função de leitura desvia do disco rígido local e chama o Pipeline JIT do Crompressor.
	// Cada leitura de Page Fault chama isso na velocidade da luz.
	data, err := FetchPage(req.Offset, req.Size)
	if err != nil {
		// Log para telemetria de SRE, mas devolve array vazio em caso de miss buffer
		log.Printf("[VFS-JIT] Page Miss no Offset %d: %v", req.Offset, err)
		resp.Data = make([]byte, req.Size)
		return nil
	}
	resp.Data = data
	return nil
}
