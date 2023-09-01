package installable

import (
	"context"
	"errors"
	"path"

	"github.com/charmbracelet/log"
	"github.com/magefile/mage/sh"
)

var goBinaryType = "go:binary"

type goBinaryOption struct {
	CI string `yaml:"ci"`
}

type goBinary struct {
	name      string
	version   string
	versioned string
	source    string
	option    goBinaryOption
}

func (a *goBinary) Install(_ context.Context, dst string) (string, error) {
	installed := path.Join(dst, a.versioned)
	if err := checkInstalled(dst, a.name, a.versioned, a.option.CI); err != nil {
		if errors.Is(err, ErrInstallableAlreadyInstalled) {
			return installed, nil
		}
		return installed, err
	}
	log.Infof("installing %s", a.versioned)

	env := map[string]string{
		"GOBIN": installed,
	}
	return installed, sh.RunWithV(env, "go", "install", a.source+"@"+a.version)
}

func (a *goBinary) Runtime() Installable {
	return nil
}
