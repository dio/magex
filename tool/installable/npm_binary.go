package installable

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/magefile/mage/sh"
)

var npmBinaryType = "npm:binary"

type npmBinaryOption struct {
	Runtime string `yaml:"runtime"`
	CI      string `yaml:"ci"`
}

type npmBinary struct {
	name      string
	version   string
	versioned string
	source    string
	runtime   Installable
	option    npmBinaryOption
}

func (a *npmBinary) Install(_ context.Context, dst string) (string, error) {
	installed := path.Join(dst, a.versioned, "node_modules", ".bin")

	if err := checkInstalled(dst, a.name, a.versioned, a.option.CI); err != nil {
		if errors.Is(err, ErrInstallableAlreadyInstalled) {
			return installed, nil
		}
		return installed, err
	}
	fmt.Printf("Installing %s", a.versioned)
	fmt.Println()

	return installed,
		sh.RunV("npm", "install", "--prefix", path.Join(dst, a.versioned), a.source+"@"+a.version)
}

func (a *npmBinary) Runtime() Installable {
	return a.runtime
}
