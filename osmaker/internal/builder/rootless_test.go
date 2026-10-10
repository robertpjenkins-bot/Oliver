package builder

import (
	"os"
	"testing"
)

// The probe must give a clear yes or no and leave nothing behind, whatever
// this machine allows.
func TestRootlessProbeCleansUp(t *testing.T) {
	before, _ := os.ReadDir(os.TempDir())
	t.Logf("rootless possible here: %v", rootlessPossible())
	after, _ := os.ReadDir(os.TempDir())
	if len(after) > len(before) {
		t.Errorf("probe left files in %s", os.TempDir())
	}
}
