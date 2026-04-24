echo "Monorepo VFS: Scan de Pacotes Workspaces"
echo "{\"workspaces\": [\"packages/*\", \"apps/*\"]}" > ./merged/pnpm-workspace.yaml
ls -la ./merged/pnpm-workspace.yaml
