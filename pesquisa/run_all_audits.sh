#!/bin/bash
# ==============================================================================
# 🚀 Crompressor Master Audit Script (V20)
# Executa todos os testes de pesquisa do 01 ao 17 sequencialmente
# ==============================================================================

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "╔══════════════════════════════════════════════════════╗"
echo "║      Mestre de Auditoria CROM (Testes 01-37)         ║"
echo "╚══════════════════════════════════════════════════════╝"

echo "🔨 Construindo Crompressor na RAIZ..."
cd ../
make clean build > /dev/null
cd "$DIR"

# 1. 01 a 05: Run Benchmarks Base
echo "▶️ Iniciando Bateria 01 a 05 (Logs, CDC, VFS, P2P, TCO)..."
cd scripts
bash run_benchmarks.sh
cd ..

# 2. 06: Imagens Experimentais (Testes Completo e Relatório)
echo "▶️ Iniciando Bateria 06 (Treinamento e Inferência Completa)..."
cd 06-image_format_analysis/scripts
# Executa todos os scripts da pesquisa de imagem sequencialmente
echo "  [06.0] Gerando Datasets..."
bash 00_generate_datasets.sh
echo "  [06.1] Treinando Cérebros..."
bash 01_train_brains.sh
echo "  [06.2] Benchmark Same-Brain..."
bash 02_same_brain_test.sh
echo "  [06.3] Benchmark Cross-Brain..."
bash 03_cross_brain_test.sh
echo "  [06.4] Testes de Inferência Pós-Treino..."
bash 04_inference_test.sh
echo "  [06.5] Teste Cérebro Universal..."
bash 05_universal_brain_test.sh
echo "  [06.6] Consolidando Relatório 06..."
bash 06_generate_report.sh
cd ../../

# 3. 07: Benchmark Comparativo (Zstd, Gzip)
echo "▶️ Iniciando Bateria 07 (Comparativo Zstd/Gzip)..."
cd 07-benchmark_comparativo
bash run_benchmark.sh
cd ..

# 4. 08: Cloud VFS Latency
echo "▶️ Iniciando Bateria 08 (Cloud VFS Latency)..."
cd 08-cloud_vfs_latency/scripts
bash 01_cloud_vs_local.sh
cd ../../

# 5. 09: Kademlia DHT Routing Efficiency 
echo "▶️ Iniciando Bateria 09 (Kademlia DHT Scale)..."
cd 09-p2p_dht_scale/scripts
bash 01_dht_routing.sh
cd ../../

# 6. 10: Entropy Shield Validation
echo "▶️ Iniciando Bateria 10 (Entropy Shield /dev/urandom)..."
cd 10-entropy_shield/scripts
bash 01_entropy_stress.sh
cd ../../

# 7. 11: Metamorphic Spawning Benchmark (V14)
echo "▶️ Iniciando Bateria 11 (Metamorphic Spawning V14)..."
cd 11-metamorphic_spawning/scripts
bash 01_spawning_benchmark.sh
cd ../../

# 8. 12: V8 Security Fuzzing (V14)
echo "▶️ Iniciando Bateria 12 (V8 Security Fuzzing)..."
cd 12-v8_security_fuzzing/scripts
bash 01_security_fuzzing.sh
cd ../../

# 9. 13: Epigenetic P2P Mesh (V14)
echo "▶️ Iniciando Bateria 13 (Epigenetic P2P Mesh)..."
cd 13-epigenetic_p2p_mesh/scripts
bash 01_p2p_mesh_test.sh
cd ../../

# 10. 14: Semantic CDC Benchmark (V15)
echo "▶️ Iniciando Bateria 14 (Semantic CDC Benchmark V15)..."
cd 14-semantic_cdc_benchmark/scripts
bash 01_semantic_benchmark.sh
cd ../../

# 11. 15: Hive-Mind Trust Attack Simulation (V15)
echo "▶️ Iniciando Bateria 15 (Hive-Mind Trust Attack V15)..."
cd 15-hivemind_trust_attack/scripts
bash 01_trust_attack.sh
cd ../../

# 12. 16: SIMD Assembly Benchmark (V20)
echo "▶️ Iniciando Bateria 16 (SIMD Assembly Benchmark V20)..."
cd 16-simd_assembly_benchmark/scripts
bash 01_simd_benchmark.sh
cd ../../

# 13. 17: Codebook Sharding OOM (V20)
echo "▶️ Iniciando Bateria 17 (Codebook Sharding OOM V20)..."
cd 17-codebook_sharding_oom/scripts
bash 01_sharding_test.sh
cd ../../

# Bateria 18: Swarm Federated Learning
echo "▶️ Iniciando Bateria 18 (Swarm Federated Learning)..."
cd 18-swarm_federated_learning/scripts
bash 01_test.sh
cd ../../

# Bateria 19: Semantic Domain Routing
echo "▶️ Iniciando Bateria 19 (Semantic Domain Routing)..."
cd 19-semantic_domain_routing/scripts
bash 01_test.sh
cd ../../

# Bateria 20: Codebook Radioactive Decay
echo "▶️ Iniciando Bateria 20 (Codebook Radioactive Decay)..."
cd 20-codebook_radioactive_decay/scripts
bash 01_test.sh
cd ../../

# Bateria 21: Temporal Shift Compression
echo "▶️ Iniciando Bateria 21 (Temporal Shift Compression)..."
cd 21-temporal_shift_compression/scripts
bash 01_test.sh
cd ../../

# Bateria 22: GPU HNSW Offload CUDA
echo "▶️ Iniciando Bateria 22 (GPU HNSW Offload CUDA)..."
cd 22-gpu_hnsw_offload_cuda/scripts
bash 01_test.sh
cd ../../

# Bateria 23: Hyper Sparse Delta Encoding
echo "▶️ Iniciando Bateria 23 (Hyper Sparse Delta Encoding)..."
cd 23-hyper_sparse_delta_encoding/scripts
bash 01_test.sh
cd ../../

# Bateria 24: Cold Storage Crystallization
echo "▶️ Iniciando Bateria 24 (Cold Storage Crystallization)..."
cd 24-cold_storage_crystallization/scripts
bash 01_test.sh
cd ../../

# Bateria 25: Zero Knowledge Codebook Poisoning
echo "▶️ Iniciando Bateria 25 (Zero Knowledge Codebook Poisoning)..."
cd 25-zero_knowledge_codebook_poisoning/scripts
bash 01_test.sh
cd ../../

# Bateria 26: Forward Error Correction P2P
echo "▶️ Iniciando Bateria 26 (Forward Error Correction P2P)..."
cd 26-forward_error_correction_p2p/scripts
bash 01_test.sh
cd ../../

# Bateria 27: Quantum Resistant GossipSub
echo "▶️ Iniciando Bateria 27 (Quantum Resistant GossipSub)..."
cd 27-quantum_resistant_gossipsub/scripts
bash 01_test.sh
cd ../../

# Bateria 28: Cross Platform Compilation
echo "▶️ Iniciando Bateria 28 (Cross Platform Compilation)..."
cd 28-cross_platform_compilation/scripts
bash 01_test.sh
cd ../../

# Bateria 29: Video Stream Delta
echo "▶️ Iniciando Bateria 29 (Video Stream Delta)..."
cd 29-video_stream_delta/scripts
bash 01_test.sh
cd ../../

# Bateria 30: Game Assets Deduplication
echo "▶️ Iniciando Bateria 30 (Game Assets Deduplication)..."
cd 30-game_assets_deduplication/scripts
bash 01_test.sh
cd ../../

# Bateria 31: OS Level Caching /dev/shm
echo "▶️ Iniciando Bateria 31 (OS Level Caching /dev/shm)..."
cd 31-os_level_caching/scripts
bash 01_test.sh
cd ../../

# Bateria 32: IPC UDS Latency Multi-App
echo "▶️ Iniciando Bateria 32 (IPC UDS Latency Multi-App)..."
cd 32-ipc_uds_latency/scripts
bash 01_test.sh
cd ../../

# Bateria 33: Satellite FEC Simulation
echo "▶️ Iniciando Bateria 33 (Satellite FEC Simulation)..."
cd 33-satellite_simulation_fec/scripts
bash 01_test.sh
cd ../../

# Bateria 34: Thermal Throttling Edge
echo "▶️ Iniciando Bateria 34 (Thermal Throttling Edge)..."
cd 34-thermal_throttling_edge/scripts
bash 01_test.sh
cd ../../

# Bateria 35: Hybrid Quantum Benchmark
echo "▶️ Iniciando Bateria 35 (Hybrid Quantum Benchmark)..."
cd 35-hybrid_quantum_bench/scripts
bash 01_test.sh
cd ../../

# Bateria 36: CUDA Sandbox Validation
echo "▶️ Iniciando Bateria 36 (CUDA Sandbox Validation)..."
cd 36-cuda_sandbox_validation/scripts
bash 01_test.sh
cd ../../

# Bateria 37: Exabyte Codebook Scale - LFU Radioactive Decay
echo "▶️ Iniciando Bateria 37 (Exabyte Codebook Scale - Radioactive Decay)..."
cd 37-exabyte_codebook_scale/scripts
bash 01_test.sh
cd ../../

echo "✅ [ALL DONE] Auditoria Padrão Exabyte Completa de 01 a 37 concluída com sucesso."
