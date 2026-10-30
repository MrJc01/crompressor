package cromlib

// noBrainChunkSize keeps the chunk table negligible: without a brain, every chunk
// is a literal and all the compression comes from zstd over the block.
const noBrainChunkSize = 1024 * 1024

// AutoPack compresses a file without a codebook. A brain only pays off when it is
// shared, so a lone file gets plain block compression and restores on its own.
func AutoPack(inputPath, outputPath string, opts PackOptions) (*Metrics, error) {
	opts.ChunkSize = noBrainChunkSize
	opts.EmbedBrain = false
	return Pack(inputPath, outputPath, "", opts)
}
