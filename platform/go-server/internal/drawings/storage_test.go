package drawings

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfiguredDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "private", "drawings")
	resolved, err := PrepareDirectory(root)
	if err != nil || !filepath.IsAbs(resolved) {
		t.Fatalf("prepare: %q %v", resolved, err)
	}
	files, err := os.ReadDir(resolved)
	if err != nil || len(files) != 0 {
		t.Fatalf("write probe left files: %v %v", files, err)
	}
	s := NewStorage(resolved)
	pdf := "%PDF-1.7\n%%EOF"
	saved, err := s.Save(strings.NewReader(pdf), "drawing.pdf")
	if err != nil {
		t.Fatal(err)
	}
	path, err := s.Path(saved.Key)
	if err != nil || filepath.Dir(filepath.Dir(path)) != resolved {
		t.Fatalf("file escaped configured root: %q %v", path, err)
	}
	// Reopening the same root must preserve preview/download access after restart.
	path, err = NewStorage(resolved).Path(saved.Key)
	data, readErr := os.ReadFile(path)
	if err != nil || readErr != nil || string(data) != pdf {
		t.Fatalf("reopen failed: %v %v", err, readErr)
	}
	if _, err = PrepareDirectory(path); err == nil {
		t.Fatal("file incorrectly accepted as storage directory")
	}
	if _, err = PrepareDirectory("  "); err == nil {
		t.Fatal("blank configuration accepted")
	}
}

func TestStorageValidation(t *testing.T) {
	s := NewStorage(t.TempDir())
	var imageData bytes.Buffer
	if e := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 10, 10))); e != nil {
		t.Fatal(e)
	}
	for _, test := range []struct {
		name  string
		data  []byte
		valid bool
	}{{"图纸.png", imageData.Bytes(), true}, {"图纸.jpg", imageData.Bytes(), false}, {"图纸.pdf", []byte("<html>bad</html>"), false}, {"图纸.pdf", []byte("%PDF-1.7\n%%EOF"), true}, {"图纸.dwg", []byte("AC1027 test"), true}, {"图纸.step", []byte("ISO-10303-21;"), true}, {"图纸.svg", []byte("<svg/>"), false}, {"空.png", nil, false}} {
		t.Run(test.name+string(test.data[:min(4, len(test.data))]), func(t *testing.T) {
			f, e := s.Save(bytes.NewReader(test.data), test.name)
			if (e == nil) != test.valid {
				t.Fatalf("valid=%t error=%v", test.valid, e)
			}
			if e == nil {
				path, e := s.Path(f.Key)
				if e != nil {
					t.Fatal(e)
				}
				got, _ := os.ReadFile(path)
				if !bytes.Equal(got, test.data) || f.Size != int64(len(test.data)) {
					t.Fatal("original file changed")
				}
			}
		})
	}
	if _, e := s.Path("../../outside"); e == nil {
		t.Fatal("path traversal allowed")
	}
	if safeName(`C:\private\drawing.png`) != "drawing.png" {
		t.Fatal("Windows filename not sanitized")
	}
	// Bound oversized streams without allocating the entire test upload.
	if _, e := s.Save(io.LimitReader(zeroReader{}, MaxFileSize+1), "big.pdf"); e == nil || !strings.Contains(e.Error(), "20MB") {
		t.Fatal("size limit not enforced", e)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }
