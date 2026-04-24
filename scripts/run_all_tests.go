package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	fmt.Println("[+] Iniciando Auditoria Completa CROM (Staff SRE Engine)")
	fmt.Println("=========================================================")

	report := bytes.NewBufferString("# Relatório de Auditoria CROM\n\n")
	report.WriteString(fmt.Sprintf("**Data da Execução:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 1. Bench Ratio
	fmt.Println("[1/4] Executando bench_ratio.sh (Eficiência / Hit Rate)...")
	report.WriteString("## 1. Bench Ratio (Eficiência de Compressão)\n\n")
	out, err := exec.Command("bash", "-c", "./scripts/tests/bench_ratio.sh").CombinedOutput()
	if err != nil {
		fmt.Printf("Falha ao rodar bench_ratio: %v\n", err)
		report.WriteString(fmt.Sprintf("❌ **Falhou:**\n```\n%v\n```\n\n", string(out)))
	} else {
		// Pega as linhas CSV e converte em Markdown
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		hasData := false
		for _, line := range lines {
			if strings.Contains(line, ",") {
				if !hasData {
					report.WriteString("| Arquivo | Tamanho Original | Tamanho CROM | \n")
					report.WriteString("|---------|------------------|--------------|\n")
					hasData = true
				}
				cols := strings.Split(line, ",")
				if len(cols) >= 3 {
					report.WriteString(fmt.Sprintf("| %s | %s | %s |\n", cols[0], cols[1], cols[2]))
				}
			}
		}
		report.WriteString("\n*O Hit Rate pode ser calculado comparando as colunas.*\n\n")
	}

	// 2. CDC Resilience
	fmt.Println("[2/4] Executando cdc_resilience.sh (Rabin Shift 1 Byte)...")
	report.WriteString("## 2. CDC Resilience (Shift de 1 Byte)\n\n")
	out, err = exec.Command("bash", "-c", "./scripts/tests/cdc_resilience.sh").CombinedOutput()
	if err != nil {
		fmt.Printf("Falha ao rodar cdc_resilience: %v\n", err)
		report.WriteString(fmt.Sprintf("❌ **Falhou:**\n```\n%v\n```\n\n", string(out)))
	} else {
		report.WriteString("```text\n")
		// Extrai apenas as linhas contendo a resposta
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, v := range lines {
			if strings.Contains(v, "Tamanho Original") || strings.Contains(v, "Tamanho Shifted") || strings.Contains(v, "Diferença Absoluta") {
				report.WriteString(v + "\n")
			}
		}
		report.WriteString("```\n\n")
	}

	// 3. Sovereignty Kill
	fmt.Println("[3/4] Executando sovereignty_kill.sh (Auto-Unmount FUSE)...")
	report.WriteString("## 3. Sovereignty Kill (Auto-Unmount FUSE)\n\n")
	out, err = exec.Command("bash", "-c", "./scripts/tests/sovereignty_kill.sh").CombinedOutput()
	if err != nil {
		fmt.Printf("Falha ao rodar sovereignty_kill: %v\n", string(out))
		report.WriteString(fmt.Sprintf("❌ **Falhou:**\n```\n%s\n```\n\n", string(out)))
	} else {
		msg := "Falha desconhecida"
		if strings.Contains(string(out), "[OK]") {
			msg = "✅ Passou: Ponto de montagem desapareceu instantaneamente em contato com a morte do codebook."
		}
		report.WriteString(msg + "\n\n")
	}

	// 4. Fuzziness Diff
	fmt.Println("[4/4] Executando fuzziness_diff.sh (O Analista de Clones LSH)...")
	report.WriteString("## 4. Fuzziness Diff (LSH Clones)\n\n")
	out, err = exec.Command("bash", "-c", "./scripts/tests/fuzziness_diff.sh").CombinedOutput()
	if err != nil {
		report.WriteString(fmt.Sprintf("❌ **Falhou:**\n```\n%v\n```\n\n", string(out)))
	} else {
		report.WriteString("```text\n")
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, v := range lines {
			if strings.Contains(v, "Tamanho:") {
				report.WriteString(v + "\n")
			}
		}
		report.WriteString("```\n\n")
	}

	fmt.Println("[+] Gravando relatório_auditoria.md...")
	err = os.WriteFile("relatorio_auditoria.md", report.Bytes(), 0644)
	if err != nil {
		fmt.Printf("Erro ao salvar relatório: %v\n", err)
		return
	}
	fmt.Println("[+] Sucesso! Analise o arquivo relatorio_auditoria.md")
}
