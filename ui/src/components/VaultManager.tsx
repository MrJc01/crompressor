import React, { useState } from 'react';

// Interfaces mapeadas para o SDK Go
interface VaultItem {
  id: string;
  name: string;
  sizeDesc: string;
  codebookPattern: string;
  isMounted: boolean;
  isEncrypted: boolean;
}

const mockVaults: VaultItem[] = [
  { id: '1', name: 'financeiro_2026.crom', sizeDesc: '85MB (Hit Rate 98%)', codebookPattern: 'documentos.cromdb', isMounted: false, isEncrypted: true },
  { id: '2', name: 'fotos_familia_vfs.crom', sizeDesc: '4GB (Hit Rate 65%)', codebookPattern: 'midia.cromdb', isMounted: true, isEncrypted: false },
];

export default function VaultManager() {
  const [vaults, setVaults] = useState<VaultItem[]>(mockVaults);

  // Invocação HTTP para o daemon Go local
  const handleToggleMount = async (vault: VaultItem) => {
    try {
      if (vault.isMounted) {
        const mount = prompt('Confirmar ponto de montagem a ejetar:', '/tmp/crom_vfs');
        if (!mount) return;
        await fetch('/api/unmount', { method: 'POST', body: JSON.stringify({ mount }) });
      } else {
        const crom_file = prompt('Caminho do arquivo .crom:', '/home/user/vault.crom');
        if (!crom_file) return;
        const codebook = prompt('Caminho do Codebook de Soberania:', '/home/user/meu.cromdb');
        if (!codebook) return;
        const mount = prompt('Ponto de Montagem VFS:', '/tmp/crom_vfs');
        if (!mount) return;
        await fetch('/api/mount', { method: 'POST', body: JSON.stringify({ crom_file, mount, codebook, key: '' }) });
      }

      setVaults(vaults.map(v => {
        if (v.id === vault.id) return { ...v, isMounted: !v.isMounted };
        return v;
      }));
    } catch (e) {
      alert("Erro ao contatar o Crompressor Daemon = " + e);
    }
  };

  return (
    <div className="flex flex-col h-full animate-fade-in">
      <header className="mb-8 flex justify-between items-center">
        <div>
          <h2 className="text-3xl font-extrabold text-white">Meus Cofres Sovereignty</h2>
          <p className="text-gray-400 mt-2">Drives Virtuais Compilados por LSH Pattern Matching.</p>
        </div>
        <button className="bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white font-bold py-2 px-6 rounded-lg shadow-[0_0_15px_rgba(79,70,229,0.5)] transition-all">
          + Criar Novo Cofre
        </button>
      </header>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {vaults.map((vault) => (
          <div key={vault.id} className="bg-gray-800 rounded-xl p-5 border border-gray-700 shadow-lg hover:shadow-indigo-500/20 transition-all group">
            <div className="flex justify-between items-start mb-4">
              <div className="w-12 h-12 rounded-lg bg-gray-700 flex items-center justify-center">
                {vault.isEncrypted ? '🔒' : '📦'}
              </div>
              <span className={`px-2 py-1 text-xs font-semibold rounded-full ${vault.isMounted ? 'bg-green-500/20 text-green-400 border border-green-500/30' : 'bg-gray-600/50 text-gray-400'}`}>
                {vault.isMounted ? 'MONTADO (FUSE)' : 'OFFLINE'}
              </span>
            </div>
            
            <h3 className="text-lg font-bold truncate text-gray-100">{vault.name}</h3>
            
            <dl className="mt-3 flex flex-col gap-1 text-sm text-gray-400">
              <div className="flex justify-between">
                <dt>Tamanho Virtual:</dt>
                <dd className="text-gray-200">{vault.sizeDesc}</dd>
              </div>
              <div className="flex justify-between">
                <dt>Cérebro LSH:</dt>
                <dd className="text-indigo-400 italic text-xs">{vault.codebookPattern}</dd>
              </div>
            </dl>

            <button 
              onClick={() => handleToggleMount(vault)}
              className={`mt-6 w-full py-2 rounded-md font-medium transition-colors ${
                vault.isMounted 
                  ? 'bg-red-500/20 text-red-500 hover:bg-red-500/30 border border-red-500/50' 
                  : 'bg-indigo-600 hover:bg-indigo-500 text-white'
              }`}
            >
              {vault.isMounted ? 'Ejetar Cofre (Unmount)' : 'Montar Cofre (VFS)'}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
