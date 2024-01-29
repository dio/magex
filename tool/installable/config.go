package installable

import (
	"errors"

	"gopkg.in/yaml.v3"
)

// ErrEntryInvalid notifies invalid entry.
var ErrEntryInvalid = errors.New("invalid entry")

// ErrEntryNotFound notifies not foun entry.
var ErrEntryNotFound = errors.New("not found entry")

type entry struct {
	Name    string      `yaml:"name"`
	Version string      `yaml:"version"`
	Source  string      `yaml:"source"`
	Type    string      `yaml:"type"`
	Option  interface{} `yaml:"option"`
}

func (e *entry) resolve(all *entries) (Installable, error) {
	switch e.Type {
	case goBinaryType:
		opt, err := typedOption[goBinaryOption](*e)
		if err != nil {
			return nil, err
		}
		return &goBinary{
			name:      e.Name,
			source:    e.Source,
			version:   e.Version,
			versioned: versioned(*e),
			option:    *opt,
		}, nil
	case httpArchiveType:
		opt, err := typedOption[httpArchiveOption](*e)
		if err != nil {
			return nil, err
		}
		return &httpArchive{
			name:      e.Name,
			source:    e.Source,
			version:   e.Version,
			versioned: versioned(*e),
			option:    *opt,
		}, nil
	case httpBinaryType:
		opt, err := typedOption[httpBinaryOption](*e)
		if err != nil {
			return nil, err
		}
		return &httpBinary{
			name:      e.Name,
			source:    e.Source,
			version:   e.Version,
			versioned: versioned(*e),
			option:    *opt,
		}, nil
	case npmBinaryType:
		opt, err := typedOption[npmBinaryOption](*e)
		if err != nil {
			return nil, err
		}
		bin := &npmBinary{
			name:      e.Name,
			source:    e.Source,
			version:   e.Version,
			versioned: versioned(*e),
			option:    *opt,
		}
		if all != nil && opt.Runtime != "" {
			// Use system runtime, e.g. node installed in the os instead of local installed binaries.
			bin.runtime, _ = all.resolve(opt.Runtime)
		}
		return bin, nil
	}
	return nil, ErrEntryInvalid
}

// entries hold tools data in a .magetools.yaml file.
type entries struct {
	Data []entry `yaml:"tools"`
}

func (e *entries) resolve(name string) (Installable, error) {
	for _, i := range e.Data {
		if i.Name == name {
			return i.resolve(e)
		}
	}
	return nil, ErrEntryNotFound
}

func typedOption[T option](e entry) (*T, error) {
	opt := new(T)
	if err := fromUntypedOption[T](e.Option, opt); err != nil {
		return nil, err
	}
	return opt, nil
}

func fromUntypedOption[T option](i interface{}, typed *T) error {
	b, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, typed)
}

func versioned(e entry) string {
	return e.Name + "@" + e.Version
}
