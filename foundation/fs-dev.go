//go:build !prod

package foundation

import (
	"embed"
	"io/fs"
	"os"
)

func GetBuildAssets(assets embed.FS, dir string) fs.FS {
	return os.DirFS(dir)
}
