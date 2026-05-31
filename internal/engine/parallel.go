package engine

import "context"

// downloadParallel is implemented in Task 10. Until then it falls back to
// single-stream so the package builds and earlier tests pass.
func downloadParallel(ctx context.Context, opt Options, meta *Meta, out string) (int64, error) {
	return downloadSingle(ctx, opt, meta, out)
}
