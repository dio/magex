package tool

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"

	"github.com/dio/magex/tool/installable"
)

// MustLoadDefault load and sets the output directory to magetools. Panics when error.
func MustLoadDefault() *Box {
	box, err := LoadDefault()
	if err != nil {
		panic(err.Error())
	}
	return box
}

// LoadDefault load and sets the output directory to magetools.
func LoadDefault() (*Box, error) {
	return Load("magetools")
}

// LoadFromFile loads installable from file.
func LoadFromFile(dir, file string) (*Box, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return LoadFromData(dir, data)
}

// LoadFromData loads installable from data.
func LoadFromData(dir string, data []byte) (*Box, error) {
	installables, err := installable.Load(data)
	if err != nil {
		return nil, err
	}

	if !path.IsAbs(dir) {
		// TODO(dio): Use git's --show-toplevel
		dir, err = filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
	}

	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(installables))
	for name := range installables {
		names = append(names, name)
	}

	return &Box{
		dir:          dir,
		names:        names,
		installables: installables,
	}, nil
}

// Load loads .magetools.yaml and put the installation destination to dir.
func Load(dir string) (*Box, error) {
	data, err := os.ReadFile(".magetools.yaml")
	if err != nil {
		return nil, err
	}
	return LoadFromData(dir, data)
}

// Box holds all information given in .magetools.yaml
type Box struct {
	dir          string
	names        []string
	installables installable.Installables
}

// RunWithOption holds option for a running tool.
type RunWithOption struct {
	Deps []string
	Env  map[string]string
}

// Output runs a tool and sends back the output.
func (b *Box) Output(ctx context.Context, name string, args ...string) (string, error) {
	return b.OutputWith(ctx, RunWithOption{}, name, args...)
}

// OutputWith runs a tools with option and sends back the output.
func (b *Box) OutputWith(ctx context.Context, opt RunWithOption, name string, args ...string) (string, error) {
	info, err := b.resolveInstallableInfo(ctx, name, opt.Deps)
	if err != nil {
		return "", err
	}
	return sh.OutputWith(opt.Env, info.Binary, args...)
}

// Run runs a tool.
func (b *Box) Run(ctx context.Context, name string, args ...string) error {
	return b.RunWith(ctx, RunWithOption{}, name, args...)
}

// RunWith runs a tool with option.
func (b *Box) RunWith(ctx context.Context, opt RunWithOption, name string, args ...string) error {
	info, err := b.resolveInstallableInfo(ctx, name, opt.Deps)
	if err != nil {
		return err
	}
	return sh.RunWithV(opt.Env, info.Binary, args...)
}

func (b *Box) resolveInstallableInfo(ctx context.Context, name string, deps []string) (installable.Info, error) {
	deps = append(deps, name)
	p, err := b.Install(ctx, deps...)
	if err != nil {
		return installable.Info{}, err
	}
	// TODO(dio): Make it multiplatform.
	_ = os.Setenv("PATH", p+":"+os.Getenv("PATH"))
	return b.installables.ResolveInfo(name)
}

// Install installs names.
func (b *Box) Install(ctx context.Context, names ...string) (string, error) {
	var paths []string
	for _, name := range names {
		name := name
		info, err := b.installables.ResolveInfo(name)
		if err != nil {
			return strings.Join(paths, ":"), err
		}
		for _, i := range info.Installers {
			p, err := i.Install(ctx, b.dir)
			paths = append(paths, p)
			if err != nil {
				baseDir := installedBaseDir(p)
				if baseDir != "" {
					_ = os.RemoveAll(installedBaseDir(p))
				}
				return strings.Join(paths, ":"), err
			}
		}
	}
	return strings.Join(dedupe(paths), ":"), nil
}

// InstallAll installs all registered installables.
func (b *Box) InstallAll(ctx context.Context) error {
	if _, err := b.Install(ctx, b.names...); err != nil {
		return err
	}
	return nil
}

func installedBaseDir(installed string) string {
	if !strings.Contains(installed, "@v") {
		return ""
	}
	if strings.Contains(filepath.Base(installed), "@v") {
		return installed
	}
	return installedBaseDir(filepath.Dir(installed))
}

func dedupe[T string | int](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
