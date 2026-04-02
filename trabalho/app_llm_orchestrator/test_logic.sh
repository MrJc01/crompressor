echo "LLM Orchestrator: Verificando Configurações de Paging GGUF"
echo "{\"model\": \"Mistral-24B-v0.1\", \"quantization\": \"Q4_K_M\", \"paging_strategy\": \"jit\"}" > ./merged/model_config.json
ls -lh ./merged/model_config.json
