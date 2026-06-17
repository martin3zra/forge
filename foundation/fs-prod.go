//go:build prod

package foundation

import (
	"embed"
	"io/fs"
)

func GetBuildAssets(assets embed.FS, dir string) fs.FS {
	f, err := fs.Sub(assets, dir)
	if err != nil {
		panic(err)
	}
	return f
}
