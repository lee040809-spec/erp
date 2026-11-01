package main

import (
	"archive/tar"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestBuildEnvironmentOnlyTargetsLinux(t *testing.T) {
	env := strings.Join(buildEnvironment([]string{"GOOS=windows", "GOARCH=arm64", "CGO_ENABLED=1", "GOCACHE=old", "GOFLAGS=-tags=other", "PATH=kept"}, "amd64", "private-cache"), "\n")
	for _, value := range []string{"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0", "GOCACHE=private-cache", "PATH=kept"} {
		if !strings.Contains(env, value) {
			t.Fatal("missing controlled build environment")
		}
	}
	for _, value := range []string{"windows", "arm64", "-tags=other", "GOCACHE=old"} {
		if strings.Contains(env, value) {
			t.Fatal("host build setting leaked")
		}
	}
}

func TestDeploymentArchiveContainsNoConfigurationFile(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, text := range deploymentFiles("28080") {
		if strings.Contains(name, ".env") {
			t.Fatal("private env file packaged")
		}
		if err := addText(tw, name, text); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(&buf)
	count := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(h.Name, "/") || strings.Contains(h.Name, "..") {
			t.Fatal("unsafe archive entry")
		}
		count++
	}
	if count != 3 {
		t.Fatal("deployment templates missing")
	}
}
