// Package runtimeinfo exposes verified runtime metadata without credentials or file paths.
package runtimeinfo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Backup struct {
	Host        string    `json:"host"`
	Database    string    `json:"database"`
	File        string    `json:"file"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	CompletedAt time.Time `json:"completed_at"`
}

// RecordBackup is called only after the SQL export has successfully completed.
func RecordBackup(manifest, file, host, database string) error {
	f, e := os.Open(file)
	if e != nil {
		return e
	}
	defer f.Close()
	info, e := f.Stat()
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 {
		return errors.New("backup file is empty or invalid")
	}
	digest := sha256.New()
	if _, e = io.Copy(digest, f); e != nil {
		return e
	}
	absolute, e := filepath.Abs(file)
	if e != nil {
		return e
	}
	row := Backup{Host: host, Database: database, File: absolute, Size: info.Size(), SHA256: hex.EncodeToString(digest.Sum(nil)), CompletedAt: info.ModTime().UTC()}
	b, e := json.MarshalIndent(row, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(manifest), 0750); e != nil {
		return e
	}
	temp, e := os.CreateTemp(filepath.Dir(manifest), "backup-status-*.tmp")
	if e != nil {
		return e
	}
	name := temp.Name()
	defer os.Remove(name)
	if _, e = temp.Write(b); e != nil {
		temp.Close()
		return e
	}
	if e = temp.Close(); e != nil {
		return e
	}
	return os.Rename(name, manifest)
}
func ReadBackup(manifest, host, database string) (Backup, error) {
	var row Backup
	f, e := os.Open(manifest)
	if e != nil {
		return row, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, 65537))
	if e != nil {
		return row, e
	}
	if len(b) > 65536 {
		return row, errors.New("backup status too large")
	}
	if e = json.Unmarshal(b, &row); e != nil {
		return row, e
	}
	digest, e := hex.DecodeString(row.SHA256)
	if e != nil || len(digest) != 32 || row.Host != host || row.Database != database || row.Size <= 0 || row.CompletedAt.IsZero() || row.CompletedAt.After(time.Now().Add(time.Minute)) {
		return row, errors.New("backup status does not match this database")
	}
	info, e := os.Stat(row.File)
	if e != nil {
		return row, e
	}
	if !info.Mode().IsRegular() || info.Size() != row.Size || !info.ModTime().Equal(row.CompletedAt) {
		return row, errors.New("backup file has changed")
	}
	// The hash was computed at export completion; do not hash a large SQL file on every page view.
	return row, nil
}
