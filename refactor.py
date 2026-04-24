import re
import os

with open('cmd/crompressor/main.go', 'r') as f:
    original_code = f.read()

funcs_to_move = [
    'daemonCmd', 'sharedDaemonCmd', 'keysCmd', 'trustCmd', 'shareCmd', 
    'mountCmd', 'cromfsCmd', 'llmVfsCmd'
]

sys_code = """//go:build !wasm
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
"""

new_main = original_code

for fn in funcs_to_move:
    # Pattern to match the whole function. It assumes the function ends with "\n}\n"
    # and has no nested "\n}\n" inside. This is usually safe enough if the file is well-formatted.
    pattern = r"func " + fn + r"\(\) \*cobra\.Command \{.*?\n\}\n"
    match = re.search(pattern, new_main, re.DOTALL)
    if match:
        sys_code += "\n" + match.group(0)
        new_main = new_main.replace(match.group(0), "")
    else:
        print(f"Function {fn} not found!")

# Remove imports from main.go that are only used in sys_commands
imports_to_remove = [
    '"github.com/MrJc01/crompressor/internal/network"',
    '"github.com/MrJc01/crompressor/internal/vfs"',
    'cvfs "github.com/MrJc01/crompressor/pkg/cromlib/vfs"',
    '"github.com/MrJc01/crompressor/internal/cromfs"',
    '"github.com/prometheus/client_golang/prometheus/promhttp"',
]

for imp in imports_to_remove:
    new_main = new_main.replace("\t" + imp + "\n", "")

with open('cmd/crompressor/sys_commands.go', 'w') as f:
    f.write(sys_code)

with open('cmd/crompressor/main.go', 'w') as f:
    f.write(new_main)

print("Done factoring main.go.")
