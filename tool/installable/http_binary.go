package installable

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
)

var httpBinaryType = "http:binary"

type httpBinaryOption struct {
	Overrides struct {
		// TODO(dio): Make it typed.
		OS     map[string]string `yaml:"os"`
		OSArch map[string]string `yaml:"osArch"`
		Arch   map[string]string `yaml:"arch"`
		Ext    map[string]string `yaml:"ext"`
	} `yaml:"overrides"`

	// TODO(dio): Have a way to set main binary and put it in a "bin" directory.
	// This is for the case when an archive doesn't have "bin" directory, or the
	// main binary is not in the "bin" directory.

	SHAs map[string]string `yaml:"shas"`

	CI string `yaml:"ci"`
}

type httpBinary struct {
	name      string
	version   string
	versioned string
	source    string
	option    httpBinaryOption
}

func (a *httpBinary) Install(ctx context.Context, dst string) (string, error) {
	versionedDir := path.Join(dst, a.versioned)
	installed := path.Join(versionedDir, "bin")

	if err := checkInstalled(dst, a.name, a.versioned, a.option.CI); err != nil {
		if err == ErrInstallableAlreadyInstalled {
			return installed, nil
		}
		return installed, err
	}

	source, err := a.expand(a.name+":url", a.source)
	if err != nil {
		return installed, err
	}
	data, _, err := readRemoteFile(ctx, source, a.versioned)
	if err != nil {
		return installed, err
	}
	fmt.Println()

	if err := a.checksum(data); err != nil {
		return installed, err
	}

	if err := os.MkdirAll(installed, os.ModePerm); err != nil {
		return installed, err
	}

	if err := os.WriteFile(path.Join(installed, a.name), data, 0o777); err != nil {
		return installed, err
	}

	return installed, ensureBinDir(versionedDir)
}

func (a *httpBinary) Runtime() Installable {
	return nil
}

func (a *httpBinary) checksum(data []byte) error {
	// TODO(dio): Add checksum.
	name := runtime.GOOS + "-" + runtime.GOARCH
	value := infer(a.option.SHAs, name, "")
	if value == "" {
		return ErrEntryInvalid
	}

	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("failed to checksum %s: %w", a.name, ErrEntryInvalid)
	}

	h := sha256.New()
	_, _ = h.Write(data)
	sum := h.Sum(nil)
	if hex.EncodeToString(sum) != parts[1] {
		return fmt.Errorf("failed to checksum %s: %w", a.name, ErrEntryInvalid)
	}
	return nil
}

func (a *httpBinary) expand(name, text string) (string, error) {
	u, err := newExpandTemplate(name).Parse(text)
	if err != nil {
		return "", err
	}
	var rendered bytes.Buffer
	if err = u.Execute(&rendered, map[string]string{
		"Version": a.version,
		"OS":      infer(a.option.Overrides.OS, runtime.GOOS, runtime.GOOS),
		"Arch":    infer(a.option.Overrides.Arch, runtime.GOARCH, runtime.GOARCH),
		"OSArch":  infer(a.option.Overrides.OSArch, runtime.GOOS+"-"+runtime.GOARCH, runtime.GOOS+"-"+runtime.GOARCH),
		"Ext":     infer(a.option.Overrides.Ext, runtime.GOOS, ".tar.gz"), // We default to .tar.gz
	}); err != nil {
		return "", err
	}
	return rendered.String(), nil
}
