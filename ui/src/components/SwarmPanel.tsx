import React, { useState, useEffect } from 'react';

interface Peer {
  id: string;
  multiaddr: string[];
  latency_ms: number;
}

export function SwarmPanel({ onClose }: { onClose: () => void }) {
  const [peers, setPeers] = useState<Peer[]>([]);
  const [running, setRunning] = useState(false);
  const [port, setPort] = useState('4001');

  const fetchPeers = async () => {
    try {
      const res = await fetch('/api/swarm/peers');
      const json = await res.json();
      if (json.success) setPeers(json.data || []);
    } catch (e) {}
  };

  useEffect(() => {
    let interval: any;
    if (running) {
      fetchPeers();
      interval = setInterval(fetchPeers, 2000);
    }
    return () => clearInterval(interval);
  }, [running]);

  const toggleSwarm = async () => {
    if (running) {
      await fetch('/api/swarm/stop', { method: 'POST' });
      setRunning(false);
      setPeers([]);
    } else {
      const res = await fetch('/api/swarm/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ port: parseInt(port), data_dir: '/tmp/swarm' })
      });
      const json = await res.json();
      if (json.success) setRunning(true);
      else alert(json.error);
    }
  };

  return (
    <div className="fixed inset-0 z-40 flex justify-end" style={{ background: 'rgba(0,0,0,0.3)', backdropFilter: 'blur(2px)' }}>
      <div className="w-96 h-full shadow-2xl flex flex-col transform transition-transform" style={{ background: 'var(--bg-secondary)', borderLeft: '1px solid var(--border)' }}>
        <div className="flex items-center justify-between px-4 py-3" style={{ borderBottom: '1px solid var(--border)', background: 'var(--bg-toolbar)' }}>
          <div className="flex items-center gap-2">
            <span className="text-xl">🌐</span>
            <h2 className="font-bold" style={{ color: 'var(--text-primary)' }}>Monitor do Enxame P2P</h2>
          </div>
          <button onClick={onClose} className="text-xl w-8 h-8 rounded hover:bg-opacity-20" style={{ color: 'var(--text-muted)' }}>×</button>
        </div>

        <div className="p-4 flex flex-col gap-4 flex-1">
          <div className="flex gap-2">
            <input 
              value={port} 
              onChange={e => setPort(e.target.value)}
              disabled={running}
              className="px-3 py-1.5 rounded border text-sm w-24" 
              style={{ background: 'var(--bg-input)', borderColor: 'var(--border)', color: 'var(--text-primary)' }} 
              placeholder="Porta"
            />
            <button 
              onClick={toggleSwarm}
              className="flex-1 px-3 py-1.5 rounded font-bold text-sm transition-colors"
              style={{ background: running ? 'var(--danger)' : 'var(--success)', color: '#fff' }}
            >
              {running ? 'Desconectar da Rede' : 'Conectar ao Enxame'}
            </button>
          </div>

          <div className="flex-1 overflow-auto rounded border" style={{ borderColor: 'var(--border)', background: 'var(--bg-primary)' }}>
            <div className="px-3 py-2 text-xs font-bold uppercase sticky top-0" style={{ background: 'var(--bg-toolbar)', color: 'var(--text-secondary)' }}>
              Peers Conectados ({peers.length})
            </div>
            {peers.length === 0 ? (
              <div className="p-4 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
                {running ? 'Aguardando descoberta de peers (mDNS/DHT)...' : 'Swarm offline. Conecte-se para ver os peers.'}
              </div>
            ) : (
              <ul className="divide-y" style={{ borderColor: 'var(--border)' }}>
                {peers.map(p => (
                  <li key={p.id} className="p-3">
                    <div className="text-xs font-mono truncate mb-1" style={{ color: 'var(--accent)' }}>{p.id}</div>
                    <div className="flex items-center justify-between text-xs" style={{ color: 'var(--text-secondary)' }}>
                      <span>Sincronizado</span>
                      <span style={{ color: 'var(--success)' }}>{p.latency_ms}ms</span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
