# 🧬 Crompressor — Benchmark Results V2

**Date:** 2026-04-05 17:35  
**Go:** go1.25.7  
**OS/Arch:** linux/amd64  
**CPUs:** 4  

## 📊 Compression Ratio & Throughput

| Dataset | Type | Original | Packed | Ratio | Pack | Unpack | Hit Rate | Entropy | SHA-256 |
|---|---|---|---|---|---|---|---|---|---|
| go_source | Código Go repetitivo com variações | 10.0 MB | 2.2 MB | **4.62x** | 7.3 MB/s | 12.8 MB/s | 0.0% | 5.1 | ✅ |
| json_api | JSON estruturado com campos repetitivos | 10.0 MB | 3.2 MB | **3.14x** | 3.2 MB/s | 13.7 MB/s | 0.0% | 5.2 | ✅ |
| server_logs | Logs de servidor com timestamps e IPs | 10.0 MB | 3.4 MB | **2.91x** | 3.9 MB/s | 16.8 MB/s | 0.0% | 5.5 | ✅ |
| mixed_config | YAML/TOML configs com seções repetidas | 5.0 MB | 1.3 MB | **3.87x** | 6.9 MB/s | 16.9 MB/s | 0.0% | 4.7 | ✅ |
| binary_headers | Headers binários + padding + structs | 10.0 MB | 2.4 MB | **4.25x** | 4.3 MB/s | 31.1 MB/s | 0.0% | 2.2 | ✅ |
| polynomial | Dados fractal (ax²+bx+c mod 256) | 1.0 MB | 1.0 MB | **1.00x** | 30.5 MB/s | 43.2 MB/s | 100.0% | 8.0 | ✅ |
| high_entropy | Dados pseudorandom (pior caso) | 10.0 MB | 10.0 MB | **1.00x** | 64.0 MB/s | 57.1 MB/s | 100.0% | 8.0 | ✅ |
| real_go_repo | O próprio código Go do crompressor | 443.2 KB | 196.0 KB | **2.26x** | 3.8 MB/s | 34.9 MB/s | 0.0% | 5.5 | ✅ |

### 🔄 Comparison vs Standard Tools

| Dataset | Crompressor | gzip -9 | zstd -19 | Best |
|---|---|---|---|---|
| go_source | 4.62x | 38.26x | 69.09x | zstd |
| json_api | 3.14x | 8.09x | 9.96x | zstd |
| server_logs | 2.91x | 6.48x | 8.45x | zstd |
| mixed_config | 3.87x | 18.13x | 25.08x | zstd |
| binary_headers | 4.25x | 24.70x | 29.04x | zstd |
| polynomial | 1.00x | 237.07x | 2896.62x | zstd |
| high_entropy | 1.00x | 1.00x | 1.00x | 🏆 CROM |
| real_go_repo | 2.26x | 3.69x | 4.35x | zstd |

## 📈 Scaling — Engine Limits

How does the engine behave as data grows?

| Size | Ratio | Pack Speed | Unpack Speed | Train Time | Hit Rate | Memory | SHA-256 |
|---|---|---|---|---|---|---|---|
| 1 MB | **2.81x** | 3.0 MB/s | 13.4 MB/s | 88ms | 0.0% | 32 MB | ✅ |
| 10 MB | **2.91x** | 6.6 MB/s | 33.3 MB/s | 136ms | 0.0% | 8 MB | ✅ |
| 50 MB | **2.92x** | 7.9 MB/s | 35.7 MB/s | 309ms | 0.0% | 17592186044412 MB | ✅ |
| 100 MB | **2.92x** | 9.0 MB/s | 37.2 MB/s | 593ms | 0.0% | 17592186044363 MB | ✅ |
| 250 MB | **2.92x** | 8.1 MB/s | 29.4 MB/s | 2.745s | 0.0% | 17592186044177 MB | ✅ |
| 500 MB | **2.93x** | 7.6 MB/s | 31.9 MB/s | 5.512s | 0.0% | 17592186043912 MB | ✅ |

**Throughput degradation:** -153.5% from 1 MB to 500 MB

## 🔀 Chunker Comparison — Fixed vs CDC vs ACAC

| Dataset | Chunker | Ratio | Hit Rate | Speed | SHA-256 |
|---|---|---|---|---|---|
| json_api | Fixed-128B | **3.13x** | 0.0% | 3.9 MB/s | ✅ |
| json_api | FastCDC | **4.10x** | 0.0% | 4.3 MB/s | ✅ |
| json_api | ACAC-newline | **1.74x** | 0.0% | 3.4 MB/s | ✅ |
| server_logs | Fixed-128B | **2.88x** | 0.0% | 3.8 MB/s | ✅ |
| server_logs | FastCDC | **3.66x** | 0.0% | 5.0 MB/s | ✅ |
| server_logs | ACAC-newline | **2.50x** | 0.0% | 4.7 MB/s | ✅ |
| go_source | Fixed-128B | **4.60x** | 0.0% | 6.7 MB/s | ✅ |
| go_source | FastCDC | **4.30x** | 0.0% | 6.9 MB/s | ✅ |
| go_source | ACAC-newline | **1.86x** | 0.0% | 3.3 MB/s | ✅ |

**Best chunker per dataset:**
- **go_source:** Fixed-128B (4.60x)
- **json_api:** FastCDC (4.10x)
- **server_logs:** FastCDC (3.66x)

## 🗂️ VFS Mount — FUSE I/O Performance

| Metric | Value |
|---|---|
| File Size | 10.0 MB |
| Mount Latency | 1.064s |
| First-Byte Latency | 196.758ms |
| VFS Sequential Read | **84.5 MB/s** |
| Direct Disk Read | 319.4 MB/s |
| Overhead | 278.0% |
| Integrity | ✅ MATCH |

## 🐳 Docker FUSE Cascade — Real System Integration

**Result:** ✅ **SUCCESS**  
**Build Time:** 3.752s  
**Container Output:**
```
🧬 CROM VFS Docker Test: SUCCESS
This app was built from a FUSE-cascaded CROM volume.
Layers: .crom → crompressor mount → OverlayFS → docker build
```

This test proves that a Docker container can be built from source files served
through a 3-layer FUSE cascade: `.crom` → CROM VFS Mount → OverlayFS → `docker build`.

## 🧮 Fractal O(1) Engine — Polynomial Pattern Analysis

| Pattern | Description | Chunk | Ratio | Hit Rate | O(1)? | SHA-256 |
|---|---|---|---|---|---|---|
| constant_zero | All zeros (a=0, b=0, c=0) | 8B | **1.00x** | 100.0% | 🧮 | ✅ |
| constant_ff | All 0xFF (a=0, b=0, c=255) | 8B | **1.00x** | 100.0% | 🧮 | ✅ |
| linear_ramp | Linear ramp: f(x) = x mod 256 (a=0, b=1, c=0) | 8B | **1.00x** | 100.0% | 🧮 | ✅ |
| quadratic_simple | Quadratic: f(x) = x² mod 256 (a=1, b=0, c=0) | 8B | **1.00x** | 100.0% | 🧮 | ✅ |
| poly_repeating_8 | 8-byte polynomial chunk repeated 128K times | 8B | **1.00x** | 100.0% | 🧮 | ✅ |
| sawtooth_16 | 16-byte sawtooth wave repeating | 16B | **1.00x** | 100.0% | 🧮 | ✅ |
| mixed_poly_noise | 70% polynomial + 30% noise (partial fractal) | 8B | **1.00x** | 100.0% | 🧮 | ✅ |

## 🔒 Integrity Verification

✅ **All tests passed SHA-256 roundtrip verification.** Every byte was reconstructed perfectly.

---

*Generated by `go run ./benchmark/`*
