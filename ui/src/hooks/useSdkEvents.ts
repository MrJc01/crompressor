import { useState, useEffect } from 'react';

// Tipagem baseada em pkg/sdk/events.go
export type SystemEvent = {
  type: string;
  payload: any;
};

export function useSdkEvents() {
  const [connectedPeers, setConnectedPeers] = useState(0);
  const [latestEvent, setLatestEvent] = useState<string | null>(null);

  useEffect(() => {
    // Conexão nativa do Browser para o Go WebServer
    const ws = new WebSocket(`ws://${window.location.host}/ws`);

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'peer_joined' || msg.type === 'peer_left') {
           // Em sistema real leríamos a contagem do payload. Mock simplificado
           setConnectedPeers(prev => msg.type === 'peer_joined' ? prev + 1 : Math.max(0, prev - 1));
        }
        if (msg.type === 'sovereignty_kill') {
           setLatestEvent('sovereignty_kill');
        }
      } catch(e) { console.error('WS Parse Error', e) }
    };

    return () => {
        ws.close();
    };
  }, []);

  return { connectedPeers, latestEvent };
}
