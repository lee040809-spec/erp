package drawings

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const MaxFileSize int64 = 20 << 20

type Storage struct{ Root string }
type StoredFile struct {
	Key, Hash, Name, MIME string
	Size                  int64
}

var storageKey = regexp.MustCompile(`^[a-f0-9]{32}$`)

func NewStorage(root string) *Storage {
	if root == "" {
		root = "data/drawings"
	}
	return &Storage{Root: root}
}
func (s *Storage) Path(key string) (string, error) {
	if !storageKey.MatchString(key) {
		return "", errors.New("无效的图纸文件标识")
	}
	return filepath.Join(s.Root, key[:2], key), nil
}
func safeName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	return strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
}
func inspect(file *os.File, name string) (string, error) {
	var head [512]byte
	n, _ := file.ReadAt(head[:], 0)
	text := string(head[:n])
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".pdf":
		if !strings.HasPrefix(text, "%PDF-") {
			return "", errors.New("文件内容不是有效的PDF")
		}
		return "application/pdf", nil
	case ".jpg", ".jpeg", ".png":
		_, _ = file.Seek(0, 0)
		cfg, format, e := image.DecodeConfig(file)
		if e != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > 100000000 {
			return "", errors.New("图片无效或像素过大，请导出较小的图纸")
		}
		if (ext == ".png" && format != "png") || (ext != ".png" && format != "jpeg") {
			return "", errors.New("图片格式与扩展名不符")
		}
		return "image/" + format, nil
	case ".dwg":
		if !strings.HasPrefix(text, "AC10") {
			return "", errors.New("DWG文件头无效")
		}
	case ".dxf":
		if !strings.HasPrefix(text, "AutoCAD Binary DXF") && !strings.Contains(text, "SECTION") {
			return "", errors.New("DXF文件头无效")
		}
	case ".step", ".stp":
		if !strings.Contains(text, "ISO-10303-21") {
			return "", errors.New("STEP文件头无效")
		}
	default:
		return "", errors.New("支持PDF、JPG、PNG、DWG、DXF、STEP/STP文件")
	}
	return "application/octet-stream", nil
}

// Stream to disk with a strict bound; never hold large uploads in Go heap memory.
func (s *Storage) Save(reader io.Reader, name string) (StoredFile, error) {
	var result StoredFile
	name = safeName(name)
	if name == "." || name == "" || len([]rune(name)) > 255 {
		return result, errors.New("图纸文件名无效或过长")
	}
	if e := os.MkdirAll(s.Root, 0700); e != nil {
		return result, e
	}
	tmp, e := os.CreateTemp(s.Root, "upload-*")
	if e != nil {
		return result, e
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	hash := sha256.New()
	n, e := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(reader, MaxFileSize+1))
	if e != nil {
		return result, e
	}
	if n == 0 || n > MaxFileSize {
		return result, errors.New("图纸不能为空且不能超过20MB")
	}
	mime, e := inspect(tmp, name)
	if e != nil {
		return result, e
	}
	var id [16]byte
	if _, e = rand.Read(id[:]); e != nil {
		return result, e
	}
	key := hex.EncodeToString(id[:])
	path, _ := s.Path(key)
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return result, e
	}
	if e = tmp.Sync(); e != nil {
		return result, e
	}
	if e = tmp.Close(); e != nil {
		return result, e
	}
	if e = os.Rename(tmp.Name(), path); e != nil {
		return result, fmt.Errorf("图纸保存失败: %w", e)
	}
	return StoredFile{Key: key, Hash: hex.EncodeToString(hash.Sum(nil)), Name: name, MIME: mime, Size: n}, nil
}
