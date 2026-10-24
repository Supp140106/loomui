package tuitest

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AssertGolden compares got against the golden file testdata/<name>.golden
// (relative to the calling package's directory). If the file does not exist
// it is created and the test passes, so new golden files bootstrap
// themselves. Everything is re-approved when the environment variable
// LOOM_UPDATE_GOLDEN is set.
func AssertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if os.Getenv("LOOM_UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(strings.TrimRight(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		t.Logf("updated golden file %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(strings.TrimRight(got, "\n")+"\n"), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		t.Logf("created golden file %s", path)
		return
	}
	if err != nil {
		t.Fatalf("read golden %s: %v", path, err)
	}
	got = strings.TrimRight(got, "\n")
	existing := strings.TrimRight(string(want), "\n")
	if existing != got {
		t.Errorf("golden mismatch for %s\ngot:\n%s\nwant:\n%s", path, got, existing)
	}
}
