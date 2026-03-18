package audit

import (
	"testing"
)

func Test_isFilterRepoAvailable(t *testing.T) {
	// Just verify it doesn't panic and returns a valid bool
	result := isFilterRepoAvailable()
	t.Logf("isFilterRepoAvailable() = %v", result)
}
