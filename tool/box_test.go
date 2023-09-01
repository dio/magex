package tool

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstalledBaseDir(t *testing.T) {
	tests := []struct {
		name    string
		baseDir string
	}{
		{"/home/ok/magetools/ok", ""},
		{"/home/ok/magetools/ok@v1", "/home/ok/magetools/ok@v1"},
		{"/home/ok/magetools/ok@v1/bin", "/home/ok/magetools/ok@v1"},
		{"/home/ok/magetools/ok@v1/node_modules/.bin", "/home/ok/magetools/ok@v1"},
	}

	for _, test := range tests {
		require.Equal(t, test.baseDir, installedBaseDir(test.name))
	}
}
