import React, { useState } from 'react';

export default function BrainLibrary() {
  const [training, setTraining] = useState(false);

  const handleTrain = async () => {
    const input = prompt('Diretório de Treinamento LSH:', '/home/user/dataset');
    if (!input) return;
    const output = prompt('Caminho para salvar o Codebook (.cromdb):', '/home/user/meu.cromdb');
    if (!output) return;

    setTraining(true);
    try {
      await fetch('/api/train', { 
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ input, output, size: 8192 })
      });
      // App.tsx handles the websocket 'train_done' event to clear UI state
    } catch(e) {
      alert("Erro de conexão com o SDK");
    }
    setTraining(false);
  };

  return (
    <div className="animate-fade-in text-gray-100">
      <h2 className="text-3xl font-extrabold mb-2">Biblioteca de Cérebros (Codebooks)</h2>
      <p className="text-gray-400 mb-8">Gerencie e treine modelos LSH universais baseados em entropia.</p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
        <div className="bg-gray-800 p-6 rounded-xl shadow-lg border border-gray-700">
          <h3 className="text-xl font-bold mb-4">🧠 Cérebro Universal (Padrão)</h3>
          <p className="text-sm text-gray-400 mb-2">Padrões genéricos treinados sobre datasets diversificados (1GB).</p>
          <div className="bg-gray-900 rounded p-3 text-xs font-mono text-green-400 mb-4">
            Hash: f8e9a2b1c3d4e5f6<br />
            LSH Targets: 8192
          </div>
          <button className="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded text-sm w-full">Exportar Manifesto P2P</button>
        </div>
      </div>

      <div className="bg-gray-800 p-6 rounded-xl border border-indigo-900/50 shadow-[0_0_20px_rgba(79,70,229,0.1)]">
        <h3 className="text-xl font-bold mb-4">Criar Novo Cérebro Específico</h3>
        <div className="flex gap-4">
          <input type="text" placeholder="/caminho/do/dataset/fotos" className="bg-gray-900 text-white rounded p-3 flex-1 border border-gray-700 focus:outline-none focus:border-indigo-500" />
          <button 
            onClick={handleTrain} 
            disabled={training}
            className="bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white px-8 rounded font-bold transition-all"
          >
            {training ? '⚙️ Treinando...' : 'Iniciar Treinamento Mágico'}
          </button>
        </div>
      </div>
    </div>
  )
}
