package vfs

import (
	"fmt"
	"log"
	"sync"
)

var (
	pageCache sync.Map // Map simples para simular L3 Cache do Crompressor
)

// FetchPage é o núcleo do Out-Of-Core Engine. Ele é invocado a cada Page Fault do SO 
// que tenta ler o arquivo virtual .gguf
func FetchPage(offset int64, size int) ([]byte, error) {
	// 1. Verificação de O(1) Hits via Hash
	// Se a máquina já bateu nesse pedaço temporal, retornamos do Cache Rápido
	cacheKey := fmt.Sprintf("%d-%d", offset, size)
	if val, ok := pageCache.Load(cacheKey); ok {
		return val.([]byte), nil
	}

	// 2. Telemetria Ativa
	// Mostrar que o VFS interceptou o bloco mágico da rede neural
	log.Printf("[CROM-VFS] JIT Paging: Interceptada Leitura [%d -> %d]", offset, offset+int64(size))

	// 3. JIT Decompression Simulado
	// Em um ambiente de SRE de produção real, o Crompressor procuraria seu Hash Table no Codebook, 
	// acharia a entrada comprimida, descompactaria (ex: ZSTD/Snappy) e retornaria.
	// Aqui provamos a interface devolvendo Null bytes ou dados preditivos 
	// suficientes apenas para não causar Kernel Panic de Null Pointers no Llama.cpp.

	data := make([]byte, size)
	
	// Simula o cabeçalho 'GGUF' mágico no início absoluto do arquivo para enganar o lhama
	if offset == 0 && size >= 8 {
		data[0] = 0x47 // G
		data[1] = 0x47 // G
		data[2] = 0x55 // U
		data[3] = 0x46 // F
		
		// GGUF Version (Little Endian uint32 = 3)
		data[4] = 0x03
		data[5] = 0x00
		data[6] = 0x00
		data[7] = 0x00
	}

	// Cacheia e Retorna
	pageCache.Store(cacheKey, data)
	return data, nil
}

// ClearCache limpa o L3 Cache simulado para testar Cold Starts
func ClearCache() {
	pageCache.Range(func(key, value interface{}) bool {
		pageCache.Delete(key)
		return true
	})
}
