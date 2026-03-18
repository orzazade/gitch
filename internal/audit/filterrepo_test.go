package audit

import (
	"testing"
)

func TestIsFilterRepoAvailable(t *testing.T) {
	// Just verify it doesn't panic and returns a valid bool
	result := IsFilterRepoAvailable()
	t.Logf("IsFilterRepoAvailable() = %v", result)
}
