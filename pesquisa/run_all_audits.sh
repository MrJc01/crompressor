#!/bin/bash
# ==============================================================================
# 🚀 Crompressor Master Audit Script (V20)
# Executa todos os testes de pesquisa do 01 ao 17 sequencialmente
# ==============================================================================

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

echo "╔══════════════════════════════════════════════════════╗"
echo "║      Mestre de Auditoria CROM (Testes 01-90)         ║"
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

# Bateria 38: LLM Weights Deduplication
echo "▶️ Iniciando Bateria 38 (LLM Weights Deduplication)..."
cd 38-llm_weights_deduplication/scripts
bash 01_test.sh
cd ../../

# Bateria 39: DNA Genomic Compression
echo "▶️ Iniciando Bateria 39 (DNA Genomic Compression)..."
cd 39-dna_genomic_compression/scripts
bash 01_test.sh
cd ../../

# Bateria 40: WebAssembly Browser Edge Node
echo "▶️ Iniciando Bateria 40 (WebAssembly Browser Edge Node)..."
cd 40-wasm_browser_edge/scripts
bash 01_test.sh
cd ../../

# Bateria 41: Blockchain Merkle Patricia Trie
echo "▶️ Iniciando Bateria 41 (Blockchain Merkle Patricia Trie)..."
cd 41-blockchain_merkle_trie/scripts
bash 01_test.sh
cd ../../

# Bateria 42: Distributed RAM Codebook Pool
echo "▶️ Iniciando Bateria 42 (Distributed RAM Codebook Pool)..."
cd 42-distributed_ram_pool/scripts
bash 01_test.sh
cd ../../

# Bateria 43: Real-time WebRTC Audio VoIP
echo "▶️ Iniciando Bateria 43 (Real-time WebRTC Audio VoIP)..."
cd 43-webrtc_voip_delta/scripts
bash 01_test.sh
cd ../../

# Bateria 44: AI Telepathy Federated Mesh
echo "▶️ Iniciando Bateria 44 (AI Telepathy Federated Mesh)..."
cd 44-ai_telepathy_federated_mesh/scripts
bash 01_test.sh
cd ../../

# Bateria 45: IPFS Core Bypass
echo "▶️ Iniciando Bateria 45 (IPFS Core Bypass)..."
cd 45-ipfs_core_bypass/scripts
bash 01_test.sh
cd ../../

# Bateria 46: Brain-Computer Interface EEG
echo "▶️ Iniciando Bateria 46 (Brain-Computer Interface EEG)..."
cd 46-brain_computer_interface_eeg/scripts
bash 01_test.sh
cd ../../

# Bateria 47: Exa-Scale PostgreSQL WAL CDC
echo "▶️ Iniciando Bateria 47 (Exa-Scale PostgreSQL WAL CDC)..."
cd 47-exa_scale_database_cdc/scripts
bash 01_test.sh
cd ../../

# Bateria 48: Holographic NeRF 3D Streaming
echo "▶️ Iniciando Bateria 48 (Holographic NeRF 3D Streaming)..."
cd 48-holographic_nerf_streaming/scripts
bash 01_test.sh
cd ../../

# Bateria 49: Post-Quantum L3 ZK-Rollups 
echo "▶️ Iniciando Bateria 49 (Post-Quantum L3 ZK-Rollups)..."
cd 49-zk_rollups_layer3/scripts
bash 01_test.sh
cd ../../

# Bateria 50: Kubernetes Zero-Gravity CNI Sockets
echo "▶️ Iniciando Bateria 50 (Kubernetes Zero-Gravity CNI)..."
cd 50-zero_gravity_kubernetes_cni/scripts
bash 01_test.sh
cd ../../

# Bateria 51: SETI Radio-Telescope Cosmic Delta
echo "▶️ Iniciando Bateria 51 (SETI Radio-Telescope Cosmic Delta)..."
cd 51-radio_telescope_seti_delta/scripts
bash 01_test.sh
cd ../../

# Bateria 52: Cyber-Robotics ROS2 DDS Swarm
echo "▶️ Iniciando Bateria 52 (Cyber-Robotics ROS2 DDS Swarm)..."
cd 52-cyber_robotics_ros2_dds/scripts
bash 01_test.sh
cd ../../

# Bateria 53: DNA Synthetic Biology Storage
echo "▶️ Iniciando Bateria 53 (DNA Synthetic Biology Storage)..."
cd 53-synthetic_biology_storage/scripts
bash 01_test.sh
cd ../../

# Bateria 54: CERN LHC Dark Matter
echo "▶️ Iniciando Bateria 54 (CERN LHC Dark Matter)..."
cd 54-cern_lhc_dark_matter/scripts
bash 01_test.sh
cd ../../

# Bateria 55: Quantum Entanglement
echo "▶️ Iniciando Bateria 55 (Quantum Entanglement State Vector)..."
cd 55-quantum_entanglement_state_vector/scripts
bash 01_test.sh
cd ../../

# Bateria 56: Tokamak Nuclear Fusion Plasma
echo "▶️ Iniciando Bateria 56 (Tokamak Nuclear Fusion Plasma)..."
cd 56-tokamak_nuclear_fusion_plasma/scripts
bash 01_test.sh
cd ../../

# Bateria 57: Brain Hologram Visual Cortex
echo "▶️ Iniciando Bateria 57 (Brain Hologram Visual Cortex)..."
cd 57-brain_hologram_visual_cortex/scripts
bash 01_test.sh
cd ../../

# Bateria 58: Global Seismograph Array Tectonics
echo "▶️ Iniciando Bateria 58 (Global Seismograph Array Tectonics)..."
cd 58-global_seismograph_array_tectonics/scripts
bash 01_test.sh
cd ../../

# Bateria 59: Nanosecond HFT Stock Market
echo "▶️ Iniciando Bateria 59 (Nanosecond HFT Stock Market)..."
cd 59-nanosecond_hft_stock_market/scripts
bash 01_test.sh
cd ../../

# Bateria 60: Neuralink Spine Motor Bypass
echo "▶️ Iniciando Bateria 60 (Neuralink Spine Motor Bypass)..."
cd 60-neuralink_spine_motor_bypass/scripts
bash 01_test.sh
cd ../../

# Bateria 61: Submarine Deep Ocean Mesh
echo "▶️ Iniciando Bateria 61 (Submarine Deep Ocean Mesh)..."
cd 61-submarine_deep_ocean_mesh/scripts
bash 01_test.sh
cd ../../

# Bateria 62: Military UAV Hive Swarm
echo "▶️ Iniciando Bateria 62 (Military UAV Hive Swarm)..."
cd 62-military_uav_hive_swarm/scripts
bash 01_test.sh
cd ../../

# Bateria 63: James Webb Exoplanet Spectra
echo "▶️ Iniciando Bateria 63 (James Webb Exoplanet Spectra)..."
cd 63-james_webb_exoplanet_spectra/scripts
bash 01_test.sh
cd ../../

# Bateria 64: Global Carbon IoT Grid
echo "▶️ Iniciando Bateria 64 (Global Carbon IoT Grid)..."
cd 64-global_carbon_iot_grid/scripts
bash 01_test.sh
cd ../../

# Bateria 65: AlphaFold Protein Morphogenesis
echo "▶️ Iniciando Bateria 65 (AlphaFold Protein Morphogenesis)..."
cd 65-alphafold_protein_morphogenesis/scripts
bash 01_test.sh
cd ../../

# Bateria 66: Asteroid Mining LiDAR Topography
echo "▶️ Iniciando Bateria 66 (Asteroid Mining LiDAR Topography)..."
cd 66-asteroid_mining_lidar_micro_g/scripts
bash 01_test.sh
cd ../../

# Bateria 67: Whole Brain Emulation Connectome
echo "▶️ Iniciando Bateria 67 (Whole Brain Emulation Connectome)..."
cd 67-whole_brain_emulation_connectome/scripts
bash 01_test.sh
cd ../../

# Bateria 68: Tachyon Predictive Time Delta
echo "▶️ Iniciando Bateria 68 (Tachyon Predictive Time Delta)..."
cd 68-tachyon_predictive_time_delta/scripts
bash 01_test.sh
cd ../../

# Bateria 69: Cybernetic Firmware OTA FEC
echo "▶️ Iniciando Bateria 69 (Cybernetic Firmware OTA FEC)..."
cd 69-cybernetic_firmware_ota_fec/scripts
bash 01_test.sh
cd ../../

# Bateria 70: Dyson Sphere Energy Routing
echo "▶️ Iniciando Bateria 70 (Dyson Sphere Energy Routing)..."
cd 70-dyson_sphere_energy_routing/scripts
bash 01_test.sh
cd ../../

echo "✅ [PHASE 1] Baterias 01-70 concluídas!"
echo ""
echo "╔══════════════════════════════════════════════════════╗"
echo "║   Singularity Extension (Testes 71-90)               ║"
echo "╚══════════════════════════════════════════════════════╝"

# Bateria 71: Antimatter Containment Field
echo "▶️ Iniciando Bateria 71 (Antimatter Containment Field)..."
cd 71-antimatter_containment_field/scripts
bash 01_test.sh
cd ../../

# Bateria 72: Multiverse Branch Deduplication
echo "▶️ Iniciando Bateria 72 (Multiverse Branch Deduplication)..."
cd 72-multiverse_branch_deduplication/scripts
bash 01_test.sh
cd ../../

# Bateria 73: Dark Energy Expansion Telemetry
echo "▶️ Iniciando Bateria 73 (Dark Energy Expansion Telemetry)..."
cd 73-dark_energy_expansion_telemetry/scripts
bash 01_test.sh
cd ../../

# Bateria 74: Neural Dream State Codec
echo "▶️ Iniciando Bateria 74 (Neural Dream State Codec)..."
cd 74-neural_dream_state_codec/scripts
bash 01_test.sh
cd ../../

# Bateria 75: Time Crystal Periodic Compression
echo "▶️ Iniciando Bateria 75 (Time Crystal Periodic Compression)..."
cd 75-time_crystal_periodic_compression/scripts
bash 01_test.sh
cd ../../

# Bateria 76: Gravitational Wave LIGO Delta
echo "▶️ Iniciando Bateria 76 (Gravitational Wave LIGO Delta)..."
cd 76-gravitational_wave_ligo_delta/scripts
bash 01_test.sh
cd ../../

# Bateria 77: Consciousness Upload Mind-File
echo "▶️ Iniciando Bateria 77 (Consciousness Upload Mind-File)..."
cd 77-consciousness_upload_mind_file/scripts
bash 01_test.sh
cd ../../

# Bateria 78: Warp Drive Alcubierre Navigation
echo "▶️ Iniciando Bateria 78 (Warp Drive Alcubierre Navigation)..."
cd 78-warp_drive_alcubierre_nav/scripts
bash 01_test.sh
cd ../../

# Bateria 79: Parallel Universe Entangled Sync
echo "▶️ Iniciando Bateria 79 (Parallel Universe Entangled Sync)..."
cd 79-parallel_universe_entangled_sync/scripts
bash 01_test.sh
cd ../../

# Bateria 80: Black Hole Hawking Radiation Log
echo "▶️ Iniciando Bateria 80 (Black Hole Hawking Radiation Log)..."
cd 80-black_hole_hawking_radiation_log/scripts
bash 01_test.sh
cd ../../

# Bateria 81: Teleportation Qubit Relay Mesh
echo "▶️ Iniciando Bateria 81 (Teleportation Qubit Relay Mesh)..."
cd 81-teleportation_qubit_relay_mesh/scripts
bash 01_test.sh
cd ../../

# Bateria 82: Galactic Civilization Type-III Grid
echo "▶️ Iniciando Bateria 82 (Galactic Civilization Type-III Grid)..."
cd 82-galactic_civilization_type3_grid/scripts
bash 01_test.sh
cd ../../

# Bateria 83: Planck Scale Spacetime Foam
echo "▶️ Iniciando Bateria 83 (Planck Scale Spacetime Foam)..."
cd 83-planck_scale_spacetime_foam/scripts
bash 01_test.sh
cd ../../

# Bateria 84: Matrioshka Brain Solar Compute
echo "▶️ Iniciando Bateria 84 (Matrioshka Brain Solar Compute)..."
cd 84-matrioshka_brain_solar_compute/scripts
bash 01_test.sh
cd ../../

# Bateria 85: Omega Point Heat Death Simulation
echo "▶️ Iniciando Bateria 85 (Omega Point Heat Death Simulation)..."
cd 85-omega_point_heat_death_sim/scripts
bash 01_test.sh
cd ../../

# Bateria 86: CRISPR Real-Time Gene Drive
echo "▶️ Iniciando Bateria 86 (CRISPR Real-Time Gene Drive)..."
cd 86-crispr_realtime_gene_drive/scripts
bash 01_test.sh
cd ../../

# Bateria 87: Von Neumann Self-Replicating Probe Swarm
echo "▶️ Iniciando Bateria 87 (Von Neumann Self-Replicator)..."
cd 87-von_neumann_self_replicator/scripts
bash 01_test.sh
cd ../../

# Bateria 88: Magnetar Pulsar Timing Array
echo "▶️ Iniciando Bateria 88 (Magnetar Pulsar Timing Array)..."
cd 88-magnetar_pulsar_timing_array/scripts
bash 01_test.sh
cd ../../

# Bateria 89: AGI Weight Matrix Compression
echo "▶️ Iniciando Bateria 89 (AGI Weight Matrix Compression)..."
cd 89-artificial_general_intelligence_weights/scripts
bash 01_test.sh
cd ../../

# Bateria 90: Superstring 11D Calabi-Yau
echo "▶️ Iniciando Bateria 90 (Superstring 11D Calabi-Yau)..."
cd 90-superstring_11d_calabi_yau/scripts
bash 01_test.sh
cd ../../

echo ""
echo "╔══════════════════════════════════════════════════════════════════╗"
echo "║  ✅ [ALL DONE] Auditoria Singularity de 01 a 90 concluída!     ║"
echo "║  🌐 Para testes WEB (91-100), execute:                         ║"
echo "║     cd pesquisa && python3 -m http.server 8091                 ║"
echo "║     Abra: http://localhost:8091/web_audit_lab.html             ║"
echo "╚══════════════════════════════════════════════════════════════════╝"

