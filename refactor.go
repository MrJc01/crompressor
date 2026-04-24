package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
)

func main() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "cmd/crompressor/main.go", nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	funcsToMove := map[string]bool{
		"daemonCmd":       true,
		"sharedDaemonCmd": true,
		"keysCmd":         true,
		"trustCmd":        true,
		"shareCmd":        true,
		"mountCmd":        true,
		"cromfsCmd":       true,
		"llmVfsCmd":       true,
	}

	var sysFuncs []ast.Decl
	var mainFuncs []ast.Decl

	for _, decl := range f.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			if funcsToMove[fn.Name.Name] {
				sysFuncs = append(sysFuncs, decl)
				continue
			}
		}
		mainFuncs = append(mainFuncs, decl)
	}

	// Remove imports
	for _, decl := range mainFuncs {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			var newSpecs []ast.Spec
			for _, spec := range genDecl.Specs {
				if impSpec, ok := spec.(*ast.ImportSpec); ok {
					path := impSpec.Path.Value
					if path == `"github.com/MrJc01/crompressor/internal/network"` ||
						path == `"github.com/MrJc01/crompressor/internal/vfs"` ||
						path == `"github.com/MrJc01/crompressor/pkg/cromlib/vfs"` ||
						path == `"github.com/MrJc01/crompressor/internal/cromfs"` ||
						path == `"github.com/prometheus/client_golang/prometheus/promhttp"` {
						continue
					}
				}
				newSpecs = append(newSpecs, spec)
			}
			genDecl.Specs = newSpecs
		}
	}

	// Write new main.go
	f.Decls = mainFuncs
	var buf bytes.Buffer
	format.Node(&buf, fset, f)
	os.WriteFile("cmd/crompressor/main.go", buf.Bytes(), 0644)

	// Write sys_commands.go
	sysTokens := token.NewFileSet()
	sysF := &ast.File{
		Name: &ast.Ident{Name: "main"},
		Decls: sysFuncs,
	}
	var sysBuf bytes.Buffer
	sysBuf.WriteString(`//go:build !wasm

package main

import (
	"context"
	"fmt"
	"io"
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

`)
	format.Node(&sysBuf, sysTokens, sysF)
	os.WriteFile("cmd/crompressor/sys_commands.go", sysBuf.Bytes(), 0644)
}
