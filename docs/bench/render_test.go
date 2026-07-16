//go:build benchrender
// +build benchrender

package bench

import "testing"

// TestRenderBenchmarks verifies benchmark markdown renders from the maintained result set.
func TestRenderBenchmarks(t *testing.T) {
	RenderBenchmarks()
}
