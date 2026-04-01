import { pipeline, env } from 'https://cdn.jsdelivr.net/npm/@huggingface/transformers@3.0.0-alpha.14';

// Em ambientes de Web Worker, models locais ficam desligados
env.allowLocalModels = false;

// Configuração para maior performance
env.backends.onnx.wasm.numThreads = 4; // Usa mais núcleos da CPU no WASM

// Engine e cache global no Worker
let generator = null;

// Escutando as chamadas da Main Thread do UI Master
self.addEventListener('message', async (event) => {
    const msg = event.data;

    if (msg.type === 'load') {
        try {
            self.postMessage({ type: 'sys_log', msg: '[CROM Engine] Iniciando montagem do motor ONNX...' });
            
            generator = await pipeline('text-generation', msg.modelId, {
                device: msg.device,
                dtype: 'q4f16',
                progress_callback: (data) => {
                    self.postMessage({ type: 'progress', data: data });
                }
            });
            
            self.postMessage({ type: 'sys_log', msg: '[CROM Engine] Motor montado com sucesso na memória livre.' });
            self.postMessage({ type: 'load_complete' });
        } catch (err) {
            self.postMessage({ type: 'error', error: err.message });
        }
    } else if (msg.type === 'generate') {
        try {
            self.postMessage({ type: 'sys_log', msg: '[CROM Engine] Prompt recebido. Descompactando tensores na CPU/GPU...' });
            self.postMessage({ type: 'sys_log', msg: '[CROM Engine] Inferência Matemática iniciada (isso pode demorar em WASM/CPU)...' });

            const results = await generator(msg.prompt, {
                max_new_tokens: 180,
                temperature: 0.7,
                do_sample: true,
                callback_function: (beams) => {
                    try {
                        // Tenta extrair o texto limpo gerado até o momento
                        const currentText = generator.tokenizer.decode(beams[0].output_token_ids, { skip_special_tokens: true });
                        self.postMessage({ type: 'stream', text: currentText });
                    } catch(e) {
                         // Ignora erro de decoding transitório
                    }
                }
            });
            
            self.postMessage({ type: 'sys_log', msg: '[CROM Tensor] Inferência finalizada com sucesso.' });
            self.postMessage({ type: 'generate_complete', text: results[0].generated_text });

        } catch (err) {
            self.postMessage({ type: 'error', error: err.message });
        }
    }
});
