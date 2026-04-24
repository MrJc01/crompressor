import React, { useState } from 'react';

export default function CloneLab() {
  const [fuzziness, setFuzziness] = useState(0.05);

  const startCloning = async () => {
    await fetch('/api/unpack_fuzz', { method: 'POST' });
    alert(`Restauração LSH ativada com Fuzziness de ${fuzziness}! O arquivo será materializado na rede neural próxima.`);
  }

  return (
    <div className="animate-fade-in text-gray-100">
      <h2 className="text-3xl font-extrabold mb-2">Laboratório de Clones CROM</h2>
      <p className="text-gray-400 mb-8">Explore o Espaço Latente (LSH) gerando variações binárias imprecisas controladas.</p>

      <div className="bg-gray-800 p-8 rounded-xl border border-gray-700 max-w-2xl mx-auto shadow-lg">
        <div className="mb-8 text-center">
          <div className="text-6xl font-black text-transparent bg-clip-text bg-gradient-to-r from-purple-400 to-pink-600">
            {fuzziness.toFixed(2)}
          </div>
          <p className="text-sm font-bold text-gray-500 uppercase tracking-widest mt-2">Divergência LSH</p>
        </div>

        <input 
          type="range" 
          min="0.00" max="0.30" step="0.01" 
          value={fuzziness} 
          onChange={(e) => setFuzziness(parseFloat(e.target.value))}
          className="w-full h-3 bg-gray-700 rounded-lg appearance-none cursor-pointer accent-pink-500 mb-8"
        />

        <div className="grid grid-cols-2 gap-4 text-sm text-gray-400 mb-8">
          <div className="bg-gray-900 p-4 rounded text-center">
            <span className="block text-xl font-bold text-white mb-1">0.00</span>
            Extração Rigorosa (Lossless)
          </div>
          <div className="bg-gray-900 p-4 rounded text-center">
            <span className="block text-xl font-bold text-pink-500 mb-1">{'>'} 0.05</span>
            Variational Clones / Falsificações
          </div>
        </div>

        <button 
          onClick={startCloning}
          className="w-full bg-gradient-to-r from-purple-600 to-pink-600 hover:from-purple-500 hover:to-pink-500 py-3 rounded-lg font-bold shadow-[0_0_20px_rgba(219,39,119,0.3)] transition-all"
        >
          Sintetizar Clone Variacional
        </button>
      </div>
    </div>
  );
}
