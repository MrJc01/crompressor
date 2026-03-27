import React, { useState, useEffect, useCallback } from 'react';
import { useTheme } from './hooks/useTheme';
import { ProgressBar } from './components/ProgressBar';
import { SwarmPanel } from './components/SwarmPanel';

interface FileEntry {
  name: string;
  path: string;
  size: number;
  modified: string;
  is_dir: boolean;
  is_crom: boolean;
  is_cromdb: boolean;
}

const tutorialSteps = [
  { icon: '📦', title: '1. Adicionar Arquivos', desc: 'Clique em "Adicionar" na toolbar para selecionar arquivos ou pastas. O CROM usa Content-Defined Chunking (CDC).' },
  { icon: '🧠', title: '2. Treinar Cérebro (Codebook)', desc: 'Treine um Codebook. O sistema extrai padrões via LSH e cria um dicionário universal para compressão.' },
  { icon: '⚡', title: '3. Comprimir (Pack)', desc: 'Selecione um arquivo e o Codebook. Mapeia dados ao padrão mais próximo atingindo ratios absurdos.' },
  { icon: '📂', title: '4. Extrair (Unpack)', desc: 'Reconstrói o original bit-a-bit. A opção de Fuzziness permite variação genética via vizinhos LSH.' },
  { icon: '🔐', title: '5. Montar Cofre (VFS)', desc: 'Monte via FUSE. Leitura O(1). Se o Codebook for removido do disco, o volume evapora automaticamente.' },
  { icon: '🌐', title: '6. Enxame (P2P Swarm)', desc: 'Conecte-se à rede GossipSub/DHT para sincronizar manifestos e arquivos.' },
];

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function App() {
  const { theme, toggle } = useTheme();
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [currentDir, setCurrentDir] = useState('');
  const [selected, setSelected] = useState<number[]>([]);
  const [loading, setLoading] = useState(false);
  const [showTutorial, setShowTutorial] = useState(false);
  const [showSwarm, setShowSwarm] = useState(false);
  const [tutStep, setTutStep] = useState(0);
  const [status, setStatus] = useState('Pronto');
  const [progress, setProgress] = useState<{ label: string, val: number, bg?: boolean }>({ label: '', val: 0 });
  const [ctxMenu, setCtxMenu] = useState<{ x: number, y: number, index: number } | null>(null);

  useEffect(() => {
    const closeCtx = () => setCtxMenu(null);
    window.addEventListener('click', closeCtx);
    return () => window.removeEventListener('click', closeCtx);
  }, []);

  const listDir = useCallback(async (dir?: string) => {
    setLoading(true);
    setSelected([]);
    try {
      const params = dir ? `?dir=${encodeURIComponent(dir)}` : '';
      const res = await fetch(`/api/list${params}`);
      const json = await res.json();
      if (json.success) {
        setFiles(json.data.files || []);
        setCurrentDir(json.data.dir);
        setStatus(`${json.data.count} itens listados`);
      } else {
        setStatus(`Erro: ${json.error}`);
      }
    } catch (e) {
      setStatus('Erro de conexão com o backend');
    }
    setLoading(false);
  }, []);

  useEffect(() => { listDir(); }, [listDir]);

  useEffect(() => {
    const ws = new WebSocket(`ws://${window.location.host}/ws`);
    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data);
        if (msg.type === 'pack_progress' || msg.type === 'train_progress') {
          // just an arbitrary visual progression based on chunks, real % would need totalBytes
          setProgress(p => ({ label: msg.type === 'pack_progress' ? 'Comprimindo...' : 'Treinando...', val: Math.min(100, p.val + 2), bg: true }));
        } else if (msg.type === 'pack_done' || msg.type === 'unpack_done' || msg.type === 'train_done' || msg.type === 'vfs_mounted' || msg.type === 'sovereignty_kill') {
          setProgress({ label: '', val: 0 });
          setStatus(`[${msg.type}] ` + (msg.payload.ratio ? `Ratio: ${msg.payload.ratio}` : 'OK!'));
          setTimeout(() => listDir(currentDir), 1000);
          if (msg.type === 'sovereignty_kill') alert('⚠️ SOVEREIGNTY KILL-SWITCH ATIVADO\n\nCodebook foi removido. Volume FUSE abortado imediatamente.');
        } else if (msg.type.includes('error')) {
          setProgress({ label: '', val: 0 });
          setStatus(`Erro: ${msg.payload}`);
          alert(`ERRO: ${msg.payload}`);
        }
      } catch {}
    };
    return () => ws.close();
  }, [currentDir, listDir]);

  const toggleSelect = (i: number) => {
    setSelected(prev => prev.includes(i) ? prev.filter(x => x !== i) : [...prev, i]);
  };

  const handlePack = async () => {
    const sel = selected.length > 0 ? files[selected[0]] : null;
    const input = prompt('Arquivo a comprimir:', sel?.path || currentDir + '/');
    if (!input) return;
    const codebook = prompt('Caminho do Codebook (.cromdb):');
    if (!codebook) return;
    const key = prompt('Senha AES-256-GCM (vazio para s/ crypto):', '') || '';
    const output = input + '.crom';
    setProgress({ label: 'Iniciando Pack...', val: 1 });
    await fetch('/api/pack', { method: 'POST', body: JSON.stringify({ input, output, codebook, key }) });
  };

  const handleUnpack = async () => {
    const sel = selected.length > 0 ? files[selected[0]] : null;
    const input = prompt('Arquivo .crom a extrair:', sel?.path || '');
    if (!input) return;
    const codebook = prompt('Caminho do Codebook (.cromdb):');
    if (!codebook) return;
    const key = prompt('Senha AES-256-GCM (se encriptado):', '') || '';
    const fuzz = parseFloat(prompt('Fuzziness (0.00 = exato):', '0.00') || '0');
    setProgress({ label: 'Extraindo...', val: 1 });
    await fetch('/api/unpack', { method: 'POST', body: JSON.stringify({ input, output: input.replace('.crom', '.restored'), codebook, key: key, fuzziness: fuzz }) });
  };

  const handleTrain = async () => {
    const input = prompt('Pasta de Treinamento LSH:', currentDir);
    if (!input) return;
    const output = prompt('Salvar Codebook Como:', currentDir + '/brain.cromdb');
    if (!output) return;
    setProgress({ label: 'Treinando Codebook...', val: 1 });
    await fetch('/api/train', { method: 'POST', body: JSON.stringify({ input, output, size: 8192 }) });
  };

  const handleVerify = async () => {
    const orig = prompt('Arquivo Original:');
    if (!orig) return;
    const rest = prompt('Arquivo Restaurado (.restored):', orig + '.restored');
    if (!rest) return;
    setStatus('Verificando integridade SHA-256...');
    const res = await fetch('/api/verify', { method: 'POST', body: JSON.stringify({ original: orig, restored: rest }) });
    const json = await res.json();
    if (json.success) alert(json.data.match ? '✅ MATCH PERFEITO! Soberania confirmada.' : '❌ FALHA! Os arquivos são diferentes.');
  };

  const handleMount = async () => {
    const sel = selected.length > 0 ? files[selected[0]] : null;
    const inputFile = prompt('Arquivo .crom para montar via FUSE:', sel?.path || '');
    if (!inputFile) return;
    const codebook = prompt('Codebook requerido para Soberania:');
    if (!codebook) return;
    const mountParam = prompt('Ponto de Montagem VFS:', '/tmp/crom_vfs');
    if (!mountParam) return;
    await fetch('/api/mount', { method: 'POST', body: JSON.stringify({ crom_file: inputFile, mount: mountParam, codebook, key: '' }) });
    setStatus(`Montado em ${mountParam}`);
  };

  const handleUnmount = async () => {
    const mountParam = prompt('Ponto de Montagem a ejetar:', '/tmp/crom_vfs');
    if (!mountParam) return;
    const res = await fetch('/api/unmount', { method: 'POST', body: JSON.stringify({ mount: mountParam }) });
    const json = await res.json();
    if (json.success) alert('Desmontado com sucesso');
  };

  const handleDelete = async () => {
    if (selected.length === 0) return alert('Selecione um arquivo');
    const f = files[selected[0]];
    if (!confirm(`Excluir permanentemente ${f.name}?`)) return;
    const res = await fetch('/api/delete', { method: 'POST', body: JSON.stringify({ path: f.path }) });
    const json = await res.json();
    json.success ? listDir(currentDir) : alert('Erro');
  };

  const handleOpen = async (path: string) => {
    await fetch('/api/open', { method: 'POST', body: JSON.stringify({ path }) });
  };

  const handleMkdir = async () => {
    const name = prompt('Nome da nova pasta:', 'Nova Pasta');
    if (!name) return;
    await fetch('/api/mkdir', { method: 'POST', body: JSON.stringify({ path: currentDir + '/' + name }) });
    listDir(currentDir);
  };

  const handleMkfile = async () => {
    const name = prompt('Nome do novo arquivo:', 'novo_arquivo.txt');
    if (!name) return;
    await fetch('/api/mkfile', { method: 'POST', body: JSON.stringify({ path: currentDir + '/' + name }) });
    listDir(currentDir);
  };

  const handleIdentity = async () => {
    const res = await fetch('/api/identity/generate');
    const json = await res.json();
    if (json.success) alert(`Novo Par Ed25519 Gerado!\n\nPublic Key:\n${json.data.public_key}\n\nO SDK agora usará essa identidade para assinar blocos.`);
  };

  return (
    <div className="flex flex-col h-screen" style={{ background: 'var(--bg-primary)', color: 'var(--text-primary)' }} onContextMenu={e => e.preventDefault()}>
      {showSwarm && <SwarmPanel onClose={() => setShowSwarm(false)} />}
      
      {ctxMenu && (
        <div className="fixed z-50 shadow-xl rounded py-1 w-48 text-sm"
             style={{ left: ctxMenu.x, top: ctxMenu.y, background: 'var(--bg-secondary)', border: '1px solid var(--border)', color: 'var(--text-primary)' }}>
          <button className="w-full text-left px-4 py-1.5 hover:bg-[var(--bg-hover)]" onClick={() => (files[ctxMenu.index].is_dir ? listDir(files[ctxMenu.index].path) : handleOpen(files[ctxMenu.index].path))}>Abrir</button>
          <div className="h-px bg-[var(--border)] my-1"></div>
          <button className="w-full text-left px-4 py-1.5 hover:bg-[var(--bg-hover)]" onClick={handlePack}>Comprimir (.crom)</button>
          <button className="w-full text-left px-4 py-1.5 hover:bg-[var(--bg-hover)]" onClick={handleUnpack}>Extrair CROM</button>
          <button className="w-full text-left px-4 py-1.5 hover:bg-[var(--bg-hover)]" onClick={handleVerify}>Verificar Hash</button>
          <div className="h-px bg-[var(--border)] my-1"></div>
          <button className="w-full text-left px-4 py-1.5 hover:bg-[var(--bg-hover)]" onClick={handleMount}>Montar FUSE</button>
          <div className="h-px bg-[var(--border)] my-1"></div>
          <button className="w-full text-left px-4 py-1.5 text-red-500 hover:bg-[var(--bg-hover)]" onClick={handleDelete}>Excluir</button>
        </div>
      )}

      {/* TITLE BAR */}
      <div className="flex items-center justify-between px-4 py-2 select-none" style={{ background: 'var(--bg-toolbar)', borderBottom: '1px solid var(--border)' }}>
        <div className="flex items-center gap-3">
          <span className="text-xl font-black" style={{ color: 'var(--accent)' }}>⬡ Crompressor</span>
          <span style={{ color: 'var(--text-muted)', fontSize: '12px' }}>v0.1.0 — WinRAR of the Future</span>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={handleIdentity} className="px-3 py-1 bg-opacity-10 rounded text-xs" style={{ color: 'var(--text-primary)', border: '1px solid var(--border)' }}>🔑 Ed25519</button>
          <button onClick={toggle} className="px-3 py-1 rounded text-sm font-medium" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)', border: '1px solid var(--border)' }}>
            {theme === 'dark' ? '☀️ Claro' : '🌙 Escuro'}
          </button>
          <button onClick={() => { setShowTutorial(true); setTutStep(0); }} className="px-3 py-1 rounded text-sm font-medium" style={{ background: 'var(--accent)', color: '#fff' }}>
            ❓ Tutorial
          </button>
        </div>
      </div>

      {/* TOOLBAR */}
      <div className="flex items-center flex-wrap gap-1 px-3 py-2" style={{ background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border)' }}>
        {[
          { icon: '📁', label: 'Nova Pasta', fn: handleMkdir },
          { icon: '📄', label: 'Novo Arquivo', fn: handleMkfile },
          { div: true },
          { icon: '📥', label: 'Adicionar', fn: handlePack },
          { icon: '📤', label: 'Extrair', fn: handleUnpack },
          { icon: '🧪', label: 'Verificar', fn: handleVerify },
          { div: true },
          { icon: '🔐', label: 'Montar VFS', fn: handleMount },
          { icon: '⏏️', label: 'Desmontar', fn: handleUnmount },
          { div: true },
          { icon: '🧠', label: 'Treinar', fn: handleTrain },
          { icon: '🌐', label: 'P2P Swarm', fn: () => setShowSwarm(true) },
          { div: true },
          { icon: '🗑️', label: 'Excluir', fn: handleDelete },
        ].map((btn, i) => (
          btn.div ? <div key={`div-${i}`} className="mx-1" style={{ width: '1px', height: '32px', background: 'var(--border)' }} /> :
          <button key={btn.label} onClick={btn.fn}
            className="flex flex-col items-center justify-center -space-y-0.5 px-3 py-1.5 rounded text-[11px] transition-all font-medium"
            style={{ color: 'var(--text-secondary)', minWidth: '60px' }}
            onMouseEnter={e => { e.currentTarget.style.background = 'var(--bg-hover)'; e.currentTarget.style.color = 'var(--text-primary)'; }}
            onMouseLeave={e => { e.currentTarget.style.background = 'transparent'; e.currentTarget.style.color = 'var(--text-secondary)'; }}>
            <span className="text-xl leading-none">{btn.icon}</span>
            <span>{btn.label}</span>
          </button>
        ))}
      </div>

      {/* ADDRESS BAR */}
      <div className="flex items-center gap-2 px-3 py-1.5" style={{ background: 'var(--bg-secondary)', borderBottom: '1px solid var(--border)' }}>
        <button onClick={() => { setCurrentDir(currentDir.split('/').slice(0, -1).join('/') || '/'); listDir(currentDir.split('/').slice(0, -1).join('/') || '/'); }} 
          className="px-2 py-1 rounded" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)', border: '1px solid var(--border)' }}>⬆</button>
        <input className="flex-1 px-3 py-1.5 rounded text-sm outline-none" style={{ background: 'var(--bg-input)', border: '1px solid var(--border)', color: 'var(--text-primary)' }}
          value={currentDir} onChange={e => setCurrentDir(e.target.value)} onKeyDown={e => e.key === 'Enter' && listDir(currentDir)} />
        <button onClick={() => listDir(currentDir)} className="px-3 py-1 rounded text-sm font-medium" style={{ background: 'var(--accent)', color: '#fff' }}>Ir</button>
      </div>

      {/* FILE TABLE */}
      <div className="flex-1 overflow-auto">
        {loading ? (
          <div className="flex items-center justify-center h-full" style={{ color: 'var(--text-muted)' }}>Carregando...</div>
        ) : (
        <table className="w-full text-sm" style={{ borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ background: 'var(--bg-toolbar)', borderBottom: '2px solid var(--border)', position: 'sticky', top: 0, zIndex: 10 }}>
              {['', 'Nome', 'Tamanho', 'Tipo', 'Modificado'].map(h => (
                <th key={h} className="text-left px-3 py-2 font-semibold select-none shadow-sm" style={{ color: 'var(--text-secondary)', fontSize: '12px' }}>{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {files.map((f, i) => (
              <tr key={i} onClick={() => toggleSelect(i)} 
                onDoubleClick={() => f.is_dir ? listDir(f.path) : handleOpen(f.path)}
                onContextMenu={(e) => { e.preventDefault(); setSelected([i]); setCtxMenu({ x: e.clientX, y: e.clientY, index: i }); }}
                className="transition-colors cursor-pointer select-none"
                style={{ background: selected.includes(i) ? 'var(--bg-selected)' : 'transparent', borderBottom: '1px solid var(--border)' }}
                onMouseEnter={e => { if (!selected.includes(i)) e.currentTarget.style.background = 'var(--bg-hover)'; }}
                onMouseLeave={e => { if (!selected.includes(i)) e.currentTarget.style.background = 'transparent'; }}>
                <td className="px-3 py-2 text-center text-lg">{f.is_dir ? '📁' : f.is_crom ? '🔒' : f.is_cromdb ? '🧠' : '📄'}</td>
                <td className="px-3 py-2 font-medium truncate" style={{ color: f.is_crom ? 'var(--accent)' : f.is_cromdb ? 'var(--success)' : 'var(--text-primary)', maxWidth: '250px' }}>{f.name}</td>
                <td className="px-3 py-2" style={{ color: 'var(--text-secondary)' }}>{f.is_dir ? 'Pasta de Arquivos' : formatSize(f.size)}</td>
                <td className="px-3 py-2" style={{ color: 'var(--text-muted)' }}>{f.is_dir ? 'Folder' : f.is_crom ? 'CROM Sovereign Archive' : f.is_cromdb ? 'LSH Codebook Brain' : 'File'}</td>
                <td className="px-3 py-2" style={{ color: 'var(--text-muted)' }}>{f.modified}</td>
              </tr>
            ))}
          </tbody>
        </table>
        )}
      </div>

      {/* STATUS BAR WITH PROGRESS */}
      <div className="flex items-center justify-between px-4 py-1.5 text-xs select-none shadow-inner" style={{ background: 'var(--bg-toolbar)', borderTop: '1px solid var(--border)', color: 'var(--text-muted)' }}>
        <div className="flex gap-4">
          <span>{files.length} itens</span>
          <span>{selected.length} selecionado</span>
        </div>
        <div className="flex gap-6 items-center">
          {progress.label ? <ProgressBar label={progress.label} progress={progress.val} total={100} /> : <span style={{ color: 'var(--text-secondary)' }}>{status}</span>}
          <span className="flex items-center gap-1 font-bold" style={{ color: 'var(--accent)' }}>
            <div className="w-2 h-2 rounded-full outline-none" style={{ background: 'var(--accent)', animation: 'pulse 2s infinite' }} /> VFS Engine
          </span>
        </div>
      </div>

      {/* TUTORIAL MODAL */}
      {showTutorial && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-60 backdrop-blur-sm">
          <div className="w-full max-w-lg rounded-xl overflow-hidden shadow-2xl" style={{ background: 'var(--bg-secondary)', border: '1px solid var(--border)' }}>
            <div className="flex items-center justify-between px-6 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
              <h2 className="text-lg font-bold" style={{ color: 'var(--accent)' }}>Como usar o Crompressor</h2>
              <button onClick={() => setShowTutorial(false)} className="text-xl text-gray-400 hover:text-white">×</button>
            </div>
            <div className="p-6 text-center" key={tutStep}>
              <div className="text-5xl mb-4">{tutorialSteps[tutStep].icon}</div>
              <h3 className="text-xl font-bold mb-3">{tutorialSteps[tutStep].title}</h3>
              <p className="text-sm leading-relaxed" style={{ color: 'var(--text-secondary)' }}>{tutorialSteps[tutStep].desc}</p>
            </div>
            <div className="flex justify-between px-6 pb-6">
              <button onClick={() => setTutStep(Math.max(0, tutStep - 1))} disabled={tutStep === 0}
                className="px-5 py-2 rounded text-sm disabled:opacity-30" style={{ background: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>← Anterior</button>
              {tutStep < tutorialSteps.length - 1 ? (
                <button onClick={() => setTutStep(tutStep + 1)} className="px-5 py-2 rounded text-sm font-bold" style={{ background: 'var(--accent)', color: '#fff' }}>Próximo →</button>
              ) : (
                <button onClick={() => setShowTutorial(false)} className="px-5 py-2 rounded text-sm font-bold" style={{ background: 'var(--success)', color: '#fff' }}>✓ Fim</button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
