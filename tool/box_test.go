package tool

import (
	"testing"
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
		if test.baseDir != installedBaseDir(test.name) {
			t.Fatalf("%s should equal %s", test.baseDir, installedBaseDir(test.name))
		}
	}
}
