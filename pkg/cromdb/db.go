//go:build !wasm

package cromdb

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// IngestFileHash popula os diretórios até o caminho apontado (mkdir -p implícito)
// e adiciona a folha final (arquivo).
func (fs *TreeFS) IngestFileHash(fullPath string, size int64, fileHash string, mode uint32) error {
	parts := strings.Split(filepath.Clean(fullPath), string(filepath.Separator))
	
	// Remove roots vazios
	if len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	parentID := int64(1) // Root ID

	// Construir arvore de subpastas se houver
	for i := 0; i < len(parts)-1; i++ {
		dirName := parts[i]
		newParent, err := fs.getOrCreateDir(parentID, dirName)
		if err != nil {
			return fmt.Errorf("failed to sync tree %s: %w", dirName, err)
		}
		parentID = newParent
	}

	// Inserir arquivo (ultimo nojeto)
	fileName := parts[len(parts)-1]
	_, err := fs.db.Exec(`
		INSERT INTO inodes (parent_id, name, is_dir, size, data_hash, mode, mod_time) 
		VALUES (?, ?, 0, ?, ?, ?, ?)
		ON CONFLICT(parent_id, name) DO UPDATE SET
			size=excluded.size, data_hash=excluded.data_hash, mode=excluded.mode
	`, parentID, fileName, size, fileHash, mode, time.Now())
	
	return err
}

func (fs *TreeFS) getOrCreateDir(parentID int64, name string) (int64, error) {
	var id int64
	err := fs.db.QueryRow("SELECT id FROM inodes WHERE parent_id = ? AND name = ? AND is_dir = 1", parentID, name).Scan(&id)
	if err == sql.ErrNoRows {
		// Dir inexiste, cria
		res, err := fs.db.Exec(`
			INSERT INTO inodes (parent_id, name, is_dir, mode, mod_time) 
			VALUES (?, ?, 1, ?, ?)
		`, parentID, name, 16877, time.Now()) // 16877 is posix 0755 | IS_DIR
		if err != nil {
			return 0, err
		}
		id, err = res.LastInsertId()
		return id, err
	}
	return id, err
}

// GetChildren retorna todos os nos filhos de um parent especifico
func (fs *TreeFS) GetChildren(parentID int64) ([]Inode, error) {
	rows, err := fs.db.Query("SELECT id, name, is_dir, size, data_hash, mode FROM inodes WHERE parent_id = ?", parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Inode
	for rows.Next() {
		var i Inode
		var nullSize sql.NullInt64
		var nullHash sql.NullString
		i.ParentID = parentID
		if err := rows.Scan(&i.ID, &i.Name, &i.IsDir, &nullSize, &nullHash, &i.Mode); err != nil {
			continue
		}
		if nullSize.Valid {
			i.Size = nullSize.Int64
		}
		if nullHash.Valid {
			i.DataHash = nullHash.String
		}
		res = append(res, i)
	}
	return res, nil
}

// LookupNode encontra um node por parentID e nme (Lookup de Dentry)
func (fs *TreeFS) LookupNode(parentID int64, name string) (*Inode, error) {
	var i Inode
	var nullSize sql.NullInt64
	var nullHash sql.NullString
	
	err := fs.db.QueryRow("SELECT id, is_dir, size, data_hash, mode FROM inodes WHERE parent_id = ? AND name = ?", parentID, name).
		Scan(&i.ID, &i.IsDir, &nullSize, &nullHash, &i.Mode)
		
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found eh nil, sem erro critico
		}
		return nil, err
	}
	
	i.ParentID = parentID
	i.Name = name
	if nullSize.Valid {
		i.Size = nullSize.Int64
	}
	if nullHash.Valid {
		i.DataHash = nullHash.String
	}
	return &i, nil
}
