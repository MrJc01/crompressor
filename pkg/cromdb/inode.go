//go:build !wasm

package cromdb

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite" // Pure go SQLite implementation to avoid CGO pains in CROM
)

// Inode representa uma pasta ou arquivo indexado no banco de dados.
type Inode struct {
	ID        int64
	ParentID  int64
	Name      string
	IsDir     bool
	Size      int64
	DataHash  string
	Mode      uint32
	ModTime   time.Time
}

// TreeFS gerencia a árvore B-Tree virtual no SQLite p/ o FUSE.
type TreeFS struct {
	db *sql.DB
}

// NewTreeFS inicializa o repositório SQLite temporário de metadados em RAM ou Disco.
// ATENÇÃO: Para `:memory:`, usamos `file::memory:?cache=shared` para que TODAS
// as conexões do pool database/sql compartilhem o MESMO banco em memória.
// Sem isso, cada conexão do pool cria um banco separado → `no such table: inodes`.
func NewTreeFS(dsn string) (*TreeFS, error) {
	if dsn == "" || dsn == ":memory:" {
		// cache=shared garante que múltiplas conexões do pool acessem o mesmo DB in-memory.
		dsn = "file::memory:?cache=shared"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Limita a 1 conexão para evitar race conditions no SQLite (single-writer).
	// Para :memory: com cache=shared, isso é redundante mas seguro.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Pragmas de performance e segurança
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA busy_timeout=5000")

	fs := &TreeFS{db: db}
	if err := fs.initSchema(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (fs *TreeFS) initSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS inodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		parent_id INTEGER,
		name TEXT NOT NULL,
		is_dir BOOLEAN NOT NULL,
		size INTEGER,
		data_hash TEXT,
		mode INTEGER,
		mod_time DATETIME,
		UNIQUE(parent_id, name)
	);
	CREATE INDEX IF NOT EXISTS idx_parent ON inodes(parent_id);
	`
	_, err := fs.db.Exec(query)
	if err != nil {
		return err
	}

	// Insere o root (/) se não existir
	var count int
	err = fs.db.QueryRow("SELECT COUNT(*) FROM inodes WHERE id = 1").Scan(&count)
	if err == nil && count == 0 {
		_, _ = fs.db.Exec(`INSERT INTO inodes (id, parent_id, name, is_dir, mode, mod_time) VALUES (1, 0, '', 1, 16877, CURRENT_TIMESTAMP)`) // 16877 = S_IFDIR | 0755
	}

	return nil
}
