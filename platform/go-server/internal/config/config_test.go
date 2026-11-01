package config

import "testing"

func TestDrawingStorageConfig(t *testing.T) {
	t.Setenv("DRAWING_STORAGE_DIR", "")
	if got := Load().DrawingStorageDir; got != "data/drawings" {
		t.Fatalf("legacy default changed: %q", got)
	}
	want := t.TempDir()
	t.Setenv("DRAWING_STORAGE_DIR", want)
	if got := Load().DrawingStorageDir; got != want {
		t.Fatalf("configured directory ignored: %q", got)
	}
}
