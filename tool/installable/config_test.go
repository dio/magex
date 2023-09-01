package installable

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveEntry(t *testing.T) {
	{
		e := &entry{
			Name:    "some",
			Type:    "some:binary",
			Version: "v1.31.0",
			Source:  "google.golang.org/protobuf/cmd/some",
			Option:  nil,
		}

		_, err := e.resolve(nil)
		require.Error(t, err)
	}

	{
		e := &entry{
			Name:    "protoc-gen-go",
			Type:    goBinaryType,
			Version: "v1.31.0",
			Source:  "google.golang.org/protobuf/cmd/protoc-gen-go",
			Option:  &goBinaryOption{},
		}

		i, err := e.resolve(nil)
		_, ok := i.(*goBinary)
		require.True(t, ok)
		require.NoError(t, err)
	}

	{
		e := &entry{
			Name:    "helm",
			Type:    httpArchiveType,
			Version: "v3.12.3",
			Source:  "https://get.helm.sh/helm-{{ .Version }}-{{ .OS }}-{{ .Arch }}{{ .Ext }}",
			Option:  &httpArchive{},
		}

		i, err := e.resolve(nil)
		_, ok := i.(*httpArchive)
		require.True(t, ok)
		require.NoError(t, err)
	}

	{
		e := &entry{
			Name:    "protoc-gen-connect-es",
			Type:    npmBinaryType,
			Version: "v0.13.0",
			Source:  "@bufbuild/protoc-gen-connect-es",
			Option: &npmBinaryOption{
				Runtime: "node",
			},
		}

		all := &entries{
			Data: []entry{
				{
					Name: "node",
					Type: httpArchiveType,
				},
			},
		}

		i, err := e.resolve(all)
		bin, ok := i.(*npmBinary)
		require.True(t, ok)
		require.NoError(t, err)
		require.NotNil(t, bin.runtime)
	}
}
