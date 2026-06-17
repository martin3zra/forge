# inertia

Bootstraps an [Inertia.js](https://inertiajs.com) server adapter (`gonertia`) wired to a Vite frontend, handling both dev (HMR) and production (manifest) modes.

## Usage

```go
//go:embed all:public
var assets embed.FS

//go:embed resources/views/root.html
var resources embed.FS

i := inertia.InitInertia(assets, resources, "5173") // port = Vite dev server port
```

`InitInertia` returns a `*gonertia.Inertia` you pass to the router / `routing.Context.Render`.

## Mode selection

- **Dev** — if `./public/hot` exists (created by `laravel-vite-plugin` in dev), it serves assets from the Vite dev server and enables HMR. The `vite` template func resolves entries against the hot URL.
- **Prod** — otherwise it reads `public/build/manifest.json` from the embedded FS, sets the asset version from the manifest, and resolves hashed asset paths under `/build/`.

SSR is enabled in both modes. The root template is `resources/views/root.html`. An empty `abilities` map is shared into templates for later injection by permission middleware.
