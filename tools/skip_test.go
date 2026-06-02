package tools

import "testing"

func TestDummyForSkipping(t *testing.T) {
	t.Skip("skip for auto-generating tools")
}
