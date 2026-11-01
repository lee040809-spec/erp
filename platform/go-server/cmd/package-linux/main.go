// package-linux creates only Linux releases. Private production defaults are
// embedded for this build, then removed from the source tree even on failure.
package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"machining-erp/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	rootFlag := flag.String("root", ".", "Go backend root (contains go.mod)")
	arch := flag.String("arch", "amd64", "Linux CPU architecture: amd64 (x86-64) or 386 (32-bit x86)")
	configFlag := flag.String("config", ".env.production", "private production configuration, relative to backend root")
	flag.Parse()
	if *arch != "amd64" && *arch != "386" {
		return errors.New("only Linux amd64 and 386 packaging is supported")
	}
	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		return err
	}
	if _, err = os.Stat(filepath.Join(root, "go.mod")); err != nil {
		return errors.New("run from platform/go-server or provide -root")
	}
	configPath := *configFlag
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(root, configPath)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		return errors.New("cannot read production configuration; check the private .env.production file")
	}
	values, err := godotenv.Unmarshal(string(content))
	if err != nil {
		return errors.New("production configuration syntax is invalid")
	}
	if err = config.ValidateProduction(values); err != nil {
		return err
	}
	front := filepath.Join(root, "..", "front", "dist")
	if _, err = os.Stat(filepath.Join(front, "index.html")); err != nil {
		return errors.New("frontend dist missing: run npm ci, npm run lint, npm test and npm run build first")
	}
	lockPath := filepath.Join(root, ".package-linux.lock")
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("another package build is active, or its lock requires inspection")
	}
	lock.Close()
	defer os.Remove(lockPath)
	generated := filepath.Join(root, "internal", "config", "production_embedded.env")
	// Do not overwrite any previous private resource left by an interrupted build.
	file, err := os.OpenFile(generated, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("cannot create private embed resource; inspect previous build residue")
	}
	defer os.Remove(generated)
	_, err = file.Write(content)
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		return errors.New("cannot prepare private embed resource")
	}
	if err = os.MkdirAll(filepath.Join(root, "bin"), 0700); err != nil {
		return err
	}
	work, err := os.MkdirTemp(filepath.Join(root, "bin"), ".linux-build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work) // work is a newly created private temp directory.
	cache := filepath.Join(work, "go-cache")
	binary := filepath.Join(work, "erp-server")
	goExe := filepath.Join(runtime.GOROOT(), "bin", "go")
	if runtime.GOOS == "windows" {
		goExe += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-buildvcs=false", "-tags", "production_embed", "-ldflags=-s -w", "-o", binary, "./cmd/server")
	cmd.Dir = root
	cmd.Env = buildEnvironment(os.Environ(), *arch, cache)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err = cmd.Run(); err != nil {
		return errors.New("Linux backend build failed")
	}
	releases := filepath.Clean(filepath.Join(root, "..", "..", "releases"))
	if err = os.MkdirAll(releases, 0700); err != nil {
		return err
	}
	name := "erp-linux-" + *arch + "-" + time.Now().UTC().Format("20060102-150405") + ".tar.gz"
	output := filepath.Join(releases, name)
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		f.Close()
		if !complete {
			os.Remove(output)
		}
	}()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err = addFile(tw, binary, "erp-server", 0700); err != nil {
		return err
	}
	if err = filepath.WalkDir(front, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("symlinks are not allowed in frontend build output")
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(front, p)
		return addFile(tw, p, "frontend/platform/"+filepath.ToSlash(rel), 0644)
	}); err != nil {
		return err
	}
	for name, data := range deploymentFiles(values["PORT"]) {
		if err = addText(tw, name, data); err != nil {
			return err
		}
	}
	if err = tw.Close(); err != nil {
		return err
	}
	if err = gz.Close(); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	complete = true
	data, err := os.Open(output)
	if err != nil {
		return err
	}
	defer data.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, data); err != nil {
		return err
	}
	fmt.Printf("Linux %s package: %s\nSHA256: %s\nProduction configuration embedded; no standalone env file included. Keep this private package out of Git.\n", *arch, output, hex.EncodeToString(hash.Sum(nil)))
	return nil
}

func buildEnvironment(current []string, arch, cache string) []string {
	out := []string{}
	for _, value := range current {
		key, _, _ := strings.Cut(value, "=")
		switch strings.ToUpper(key) {
		case "GOOS", "GOARCH", "CGO_ENABLED", "GOCACHE", "GOFLAGS", "GOENV":
			continue
		}
		out = append(out, value)
	}
	return append(out, "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0", "GOCACHE="+cache, "GOFLAGS=", "GOENV=off")
}

func addFile(tw *tar.Writer, source, name string, mode int64) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("package input is not a regular file")
	}
	if err = tw.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: info.Size()}); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

func addText(tw *tar.Writer, name, text string) error {
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(text))}); err != nil {
		return err
	}
	_, err := io.WriteString(tw, text)
	return err
}

func deploymentFiles(port string) map[string]string {
	return map[string]string{
		"erp.service":    "[Unit]\nDescription=Machining ERP\nAfter=network-online.target\nWants=network-online.target\n[Service]\nUser=erp\nGroup=erp\nWorkingDirectory=/opt/erp\nExecStart=/opt/erp/erp-server\nEnvironment=GO_ENV=production\nUMask=0077\nRestart=on-failure\nRestartSec=5\n[Install]\nWantedBy=multi-user.target\n",
		"nginx-erp.conf": "server {\n listen 80;\n server_name _;\n root /opt/erp/frontend;\n client_max_body_size 25m;\n location = / { return 302 /platform/; }\n location = /platform { return 302 /platform/; }\n location /platform/ { try_files $uri $uri/ /platform/index.html; }\n location /api/ { proxy_pass http://127.0.0.1:" + port + "; proxy_set_header Host $host; proxy_set_header X-Real-IP $remote_addr; proxy_set_header X-Forwarded-Proto $scheme; }\n location ~ /\\. { deny all; }\n}\n",
		"INSTALL.txt":    "Linux-only package. The erp-server binary contains PRIVATE database/SMTP configuration. Do not upload this package or binary to Git.\nExtract into /opt/erp; create a non-login service account named erp, and give it read/execute access to the binary.\nCreate the configured drawing directory and grant only the erp service user read/write access. Existing drawings must be copied separately; they are not in this archive.\nDatabase schema/data are not initialized by this package. Back up and check migrations before starting against an existing database.\nCopy frontend into a directory readable by nginx; this example uses /opt/erp/frontend/platform. Keep the binary private to the service user.\nReview erp.service and nginx-erp.conf for your actual paths and existing sites before installing them. Do not replace the full nginx configuration blindly.\nAfter installing the service unit: systemctl daemon-reload; systemctl enable --now erp. Check journalctl -u erp and /api/platform/health.\nThe binary starts from embedded production defaults without an env file. Optional .env.production in the service working directory overrides embedded keys; explicit process environment has highest priority. Development .env is never loaded in production.\nChanging embedded secrets requires rebuilding and replacing the binary; external .env.production can override without rebuilding.\n",
	}
}
