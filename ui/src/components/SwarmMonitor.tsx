import React from 'react';
import { useSdkEvents } from '../hooks/useSdkEvents';

export default function SwarmMonitor() {
  const { connectedPeers } = useSdkEvents();

  return (
    <div className="animate-fade-in text-gray-100">
      <h2 className="text-3xl font-extrabold mb-2">Monitor do Enxame (P2P Swarm)</h2>
      <p className="text-gray-400 mb-8">Visão em tempo real dos nós Soberanos sincronizados via libp2p GossipSub.</p>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Painel Estatístico */}
        <div className="col-span-1 bg-gray-800 p-6 rounded-xl border border-gray-700 flex flex-col items-center justify-center shadow-lg">
          <div className="relative">
            <div className="absolute inset-0 bg-indigo-500 blur-[50px] opacity-20 rounded-full"></div>
            <div className={`text-8xl font-black ${connectedPeers > 0 ? 'text-indigo-400' : 'text-gray-600'} transition-colors relative z-10`}>
              {connectedPeers}
            </div>
          </div>
          <p className="text-gray-400 mt-4 text-center font-bold tracking-widest uppercase text-sm">Peers Conectados</p>
        </div>

        {/* Mapa Log (Mock) */}
        <div className="col-span-2 bg-gray-900 p-6 rounded-xl border border-gray-700 shadow-inner overflow-hidden flex flex-col">
          <h3 className="text-xs font-mono text-gray-500 uppercase tracking-widest mb-4">GossipSub Live Telemetry</h3>
          
          <div className="flex-1 font-mono text-xs text-green-400 flex flex-col gap-2">
            <div>[SWARM] Buscando DHT via Bootstrap nodes...</div>
            <div>[SWARM] MDNS Local discovery ativo...</div>
            {connectedPeers > 0 && Array.from({length: connectedPeers}).map((_, i) => (
               <div className="animate-fade-in text-blue-300" key={i}>
                 {`[SYNC] Peer <12D3KooW${Math.floor(Math.random() * 99999999)}> conectou. Baixando Manifesto CROM...`}
               </div>
            ))}
            {connectedPeers === 0 && <div className="text-gray-500">Nenhum par encontrado no momento no anel soberano.</div>}
          </div>
        </div>
      </div>
    </div>
  )
}
