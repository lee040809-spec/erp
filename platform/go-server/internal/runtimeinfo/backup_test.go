package runtimeinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupStatus(t *testing.T) {
	dir := t.TempDir()
	file, status := filepath.Join(dir, "test.sql"), filepath.Join(dir, "status.json")
	if e := os.WriteFile(file, []byte("fixture backup"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := RecordBackup(status, file, "cloud", "test"); e != nil {
		t.Fatal(e)
	}
	if _, e := ReadBackup(status, "cloud", "test"); e != nil {
		t.Fatal(e)
	}
	if _, e := ReadBackup(status, "other", "test"); e == nil {
		t.Fatal("wrong server accepted")
	}
	if _, e := ReadBackup(status, "cloud", "other"); e == nil {
		t.Fatal("wrong database accepted")
	}
	if e := os.WriteFile(file, []byte("changed fixture backup"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := ReadBackup(status, "cloud", "test"); e == nil {
		t.Fatal("changed file accepted")
	}
	if e := os.Remove(file); e != nil {
		t.Fatal(e)
	}
	if _, e := ReadBackup(status, "cloud", "test"); e == nil {
		t.Fatal("missing backup accepted")
	}
}
