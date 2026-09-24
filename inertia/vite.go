package inertia

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	inertia "github.com/romsar/gonertia/v3"
)

func InitInertia(assets, resources embed.FS, port string) *inertia.Inertia {
	viteHotFile := "./public/hot"
	rootViewFile := "resources/views/root.html"

	// check if laravel-vite-plugin is running in dev mode (it puts a "hot" file in the public folder)
	_, err := os.Stat(viteHotFile)
	if err == nil {
		var opts []inertia.Option
		if ssrEnabled() {
			opts = append(opts, inertia.WithSSR())
		}
		i, err := inertia.NewFromFile(rootViewFile, opts...)

		if err != nil {
			log.Fatal(err)
		}
		i.ShareTemplateFunc("vite", func(entry string) (string, error) {
			content, err := os.ReadFile(viteHotFile)
			if err != nil {
				return "", err
			}
			url := strings.TrimSpace(string(content))
			if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
				url = url[strings.Index(url, ":")+1:]
			} else {
				url = fmt.Sprintf("//127.0.0.1:%s", port)
			}
			if entry != "" && !strings.HasPrefix(entry, "/") {
				entry = "/" + entry
			}
			return url + entry, nil
		})

		i.ShareTemplateData("hmr", true)

		// set empty variable name, later to be inject from the middleware.
		i.ShareTemplateData("abilities", map[string]bool{})

		return i
	}

	manifestPath := "public/build/manifest.json"
	opts := []inertia.Option{inertia.WithVersionFromFileFS(assets, manifestPath)}
	if ssrEnabled() {
		opts = append(opts, ssrOptions()...)
	}
	i, err := inertia.NewFromFileFS(resources, rootViewFile, opts...)
	if err != nil {
		log.Fatal(err)
	}

	manifestFile, _ := assets.Open("public/build/manifest.json")
	i.ShareTemplateFunc("vite", vite(manifestFile, "/build/"))

	return i
}

// ssrEnabled reports whether server-side rendering is on. INERTIA_SSR=false
// (or 0, off, no) turns it off: pages then render client-side only and no
// Node sidecar is needed. Unset or anything else keeps it on.
func ssrEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("INERTIA_SSR"))) {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

// ssrOptions enables SSR in production, with its failures logged and its
// render round-trip capped.
func ssrOptions() []inertia.Option {
	return []inertia.Option{
		inertia.WithSSR(),
		// Surface SSR failures (sidecar down, a page component throwing during
		// renderToString, render timeout) on the standard logger — gonertia's
		// default is io.Discard, which silently swallows every one. The request
		// still succeeds via the client-side fallback; this only makes the
		// degradation visible. Production branch only: the dev branch above has
		// no SSR sidecar under `dev.sh`, so logging there would be pure
		// connection-refused noise on every full page load.
		inertia.WithLogger(log.Default()),
		// gonertia's default SSR client is &http.Client{} — no timeout. A wedged
		// but still-listening node process would otherwise block every full-page
		// render indefinitely (Restart=on-failure does not catch a hung-alive
		// process). Cap the render round-trip; on timeout gonertia falls back to
		// client-side rendering and the logger above records it.
		inertia.WithSSRHTTPClient(&http.Client{Timeout: 2 * time.Second}),
	}
}

func vite(f fs.File, buildDir string) func(path string) (string, error) {

	defer f.Close()

	viteAssets := make(map[string]*struct {
		File   string `json:"file"`
		Source string `json:"src"`
	})
	err := json.NewDecoder(f).Decode(&viteAssets)

	if err != nil {
		log.Fatalf("cannot unmarshal vite manifest file to json: %s", err)
	}

	return func(p string) (string, error) {
		if val, ok := viteAssets[p]; ok {
			return path.Join("/", buildDir, val.File), nil
		}
		return "", fmt.Errorf("asset %q not found", p)
	}
}
