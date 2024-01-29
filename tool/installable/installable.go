package installable

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrInstallableAlreadyInstalled notifies already installed.
var ErrInstallableAlreadyInstalled = errors.New("already installed")

// Load loads all installables.
func Load(data []byte) (Installables, error) {
	loaded := new(entries)
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return nil, err
	}
	installables := make(Installables, len(loaded.Data))
	for _, e := range loaded.Data {
		resolved, err := e.resolve(loaded)
		if err != nil {
			return nil, ErrEntryInvalid
		}
		installables[e.Name] = resolved
	}
	return installables, nil
}

// Installables is a map of name to installable.
type Installables map[string]Installable

// ResolveInfo resolves info about an installable.
func (i Installables) ResolveInfo(name string) (Info, error) {
	info := Info{Key: name, Binary: name}
	if strings.Contains(name, ":") {
		parts := strings.Split(name, ":")
		if len(parts) != 2 {
			return info, fmt.Errorf("name: %s %w", name, ErrEntryInvalid)
		}
		info.Key = parts[0]
		info.Binary = parts[1]
	}

	installer, ok := i[info.Key]
	if !ok {
		return info, fmt.Errorf("unknown name: %s %w", name, ErrEntryInvalid)
	}
	if installer.Runtime() != nil {
		info.Installers = append(info.Installers, installer.Runtime())
	}
	info.Installers = append(info.Installers, installer)
	return info, nil
}

// Installable gives signature of an installable.
type Installable interface {
	Install(context.Context, string) (string, error)
	Runtime() Installable
}

// Info provides list of installers of a Key.
type Info struct {
	Key        string
	Binary     string
	Installers []Installable
}

// option is possible options for an installable.
type option interface {
	goBinaryOption | httpArchiveOption | httpBinaryOption | npmBinaryOption
}

func checkInstalled(dir, prefix, current, ci string) error {
	if ci == "skip" && os.Getenv("CI") == "true" {
		return ErrInstallableAlreadyInstalled
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == current { // TODO(dio): Check content.
			return ErrInstallableAlreadyInstalled
		}

		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix+"@") {
			// TODO(dio): Remove this when we need/allow multiple versions. Note that we also need
			// to have a querier (for sorting paths with the right order, or pointing to the right binary).
			return os.RemoveAll(path.Join(dir, entry.Name()))
		}
	}
	return nil
}
