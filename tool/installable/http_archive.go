package installable

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/codeclysm/extract/v3"
	"github.com/dustin/go-humanize"
)

var runningOnCI = os.Getenv("CI") != ""
var httpArchiveType = "http:archive"

type httpArchiveOption struct {
	StripPrefix string `yaml:"stripPrefix"`

	Overrides struct {
		// TODO(dio): Make it typed.
		OS     map[string]string `yaml:"os"`
		Arch   map[string]string `yaml:"arch"`
		Ext    map[string]string `yaml:"ext"`
		OSArch map[string]string `yaml:"osArch"`
	} `yaml:"overrides"`

	// TODO(dio): Have a way to set main binary and put it in a "bin" directory.
	// This is for the case when an archive doesn't have "bin" directory, or the
	// main binary is not in the "bin" directory.

	SHAs map[string]string `yaml:"shas"`

	CI string `yaml:"ci"`
}

type httpArchive struct {
	name      string
	version   string
	versioned string
	source    string
	option    httpArchiveOption
}

func (a *httpArchive) Install(ctx context.Context, dst string) (string, error) {
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

	if err = a.checksum(data); err != nil {
		return installed, err
	}

	br := bufio.NewReader(bytes.NewBuffer(data))
	prefix, err := a.expand(a.name+":stripPrefix", a.option.StripPrefix)
	if err != nil {
		return installed, err
	}

	if err = extract.Archive(ctx, br, versionedDir, func(s string) string {
		return strings.TrimPrefix(s, prefix)
	}); err != nil {
		return installed, err
	}

	return installed, ensureBinDir(versionedDir)
}

func (a *httpArchive) Runtime() Installable {
	return nil
}

func hasBinDir(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() == "bin" {
			return true, nil
		}
	}
	return false, nil
}

func (a *httpArchive) checksum(data []byte) error {
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
	encoded := hex.EncodeToString(sum)
	if encoded != parts[1] {
		return fmt.Errorf("failed to checksum %q: %s vs. %s %w", a.name, encoded, parts[1], ErrEntryInvalid)
	}
	return nil
}

// newExpandTemplate creates a new named template with common custom functions.
var newExpandTemplate = func(name string) *template.Template {
	return template.New(name).Funcs(template.FuncMap{
		// "trimV" is so commonly used. This makes e.g. v0.0.1 -> 0.0.1.
		"trimV": func(ver string) string {
			if strings.HasPrefix(ver, "v") {
				return ver[1:]
			}
			return ver
		},
	})
}

func (a *httpArchive) expand(name, text string) (string, error) {
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

func infer(m map[string]string, ref, fallback string) string {
	for k, v := range m {
		if k == ref {
			return v
		}
	}
	return fallback
}

func readRemoteFile(ctx context.Context, url, name string) ([]byte, http.Header, error) {
	var buf bytes.Buffer
	out := bufio.NewWriter(&buf)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, resp.Header, fmt.Errorf("unexpected status code while reading %s: %v", url, resp.StatusCode)
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, &writeCounter{name: name}))
	if err != nil {
		return nil, resp.Header, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	return buf.Bytes(), resp.Header, nil
}

func ensureBinDir(dir string) error {
	hasBin, err := hasBinDir(dir)
	if err != nil {
		return err
	}

	if !hasBin {
		_ = os.MkdirAll(path.Join(dir, "bin"), os.ModePerm)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			_ = os.Rename(path.Join(dir, entry.Name()), path.Join(dir, "bin", entry.Name()))
		}
	}

	return nil
}

type writeCounter struct {
	name  string
	total uint64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += uint64(n)
	wc.printProgress()
	return n, nil
}

func (wc writeCounter) printProgress() {
	if runningOnCI {
		return
	}
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Fprintf(os.Stderr, "\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Fprintf(os.Stderr, "\rDownloading %s... %s ", wc.name, humanize.Bytes(wc.total))
}
