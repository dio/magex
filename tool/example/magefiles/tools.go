//go:build mage

package main

import (
	"context"
	"strings"

	"github.com/magefile/mage/mg"

	"github.com/dio/magex/tool"
)

var box *tool.Box

func toolbox() *tool.Box {
	if box == nil {
		box = tool.MustLoadDefault()
	}
	return box
}

// Tools holds all tools target.
type Tools mg.Namespace

// All downloads all defined tools in .magetools.yaml.
func (Tools) All(ctx context.Context) error {
	return toolbox().InstallAll(ctx)
}

// Run enables to run a locally installed tool.
// Caveat: need to "wrap" the args with ” or "". For example: mage tools:run buf '--version'.
func (Tools) Run(ctx context.Context, name, rest string) error {
	return toolbox().Run(ctx, name, strings.Split(rest, " ")...)
}
