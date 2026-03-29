# 🧪 Pesquisa e Viabilidade Técnica: Crompressor V3

Este arquivo é o ponto central de auditoria para todos os experimentos de compressão soberana.

---

## 📂 Mapa de Documentação Técnica

- 📄 **[Manifesto de Auditoria](./Manifesto.md)**
- 📄 [01 Logs Redundancia](./01-logs_redundancia/relatorio.md)
- 📄 [02 Delta Sync Cdc](./02-delta_sync_cdc/relatorio.md)
- 📄 [03 Vfs Mount Perf](./03-vfs_mount_perf/relatorio.md)
- 📄 [04 P2p Soberania](./04-p2p_soberania/relatorio.md)
- 📄 [05 Tco Storage Frio](./05-tco_storage_frio/relatorio.md)
- 📄 [06 Image Format Analysis](./06-image_format_analysis/relatorio.md)
- 📄 [07 Benchmark Comparativo](./07-benchmark_comparativo/Relatorio_Gzip_Zstd_Crompressor.md)

---

## 🛠️ Comandos de Atualização
Para regenerar este dossiê, execute:
```bash
cd pesquisa
./setup_pesquisa.sh
```

> [!CAUTION]
> **Atenção**: Este repositório é auditado bit-a-bit. A modificação manual dos relatórios sem a execução dos benchmarks correspondentes invalida a conformidade SRE.
