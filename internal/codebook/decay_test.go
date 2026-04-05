package codebook

import (
	"testing"
	"time"
)

func TestRadioactiveDecay(t *testing.T) {
	// Mock Codebook Reader
	r := &Reader{
		lruCache: make(map[uint64][]byte),
	}
	r.lruCache[42] = []byte("universal_pattern_A")
	r.lruCache[99] = []byte("universal_pattern_B")

	engine := NewDecayEngine(r)
	engine.Touch(42)
	engine.Touch(99)

	if len(r.lruCache) != 2 {
		t.Fatalf("Cache inicial deveria ter 2 itens, tem %d", len(r.lruCache))
	}

	// Forçar chave '42' a espirar simulando que foi acessada há 11 segundos
	engine.heatmap[42] = time.Now().Add(-11 * time.Second).Unix()
	engine.heatmap[99] = time.Now().Unix() // 99 é quente

	// Disparar o Expurgo com janela de 10 segundos
	engine.decay(10 * time.Second)

	if len(r.lruCache) != 1 {
		t.Fatalf("Cache deveria ter 1 item (expurgo falhou), tem %d", len(r.lruCache))
	}

	if _, ok := r.lruCache[99]; !ok {
		t.Fatalf("O chunk quente 99 deveria ter sobrevivido ao decaimento")
	}

	if _, ok := r.lruCache[42]; ok {
		t.Fatalf("O chunk frio 42 deveria ter sido expurgado")
	}
}
