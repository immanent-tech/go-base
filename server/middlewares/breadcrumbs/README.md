# pagebreadcrumbs

A small Go package that maintains a per-session trail of page URLs as a user navigates your site, using
[chi](https://github.com/go-chi/chi) for routing and [alexedwards/scs](https://github.com/alexedwards/scs) for session
storage.

## How it works

A middleware inspects the `Referer` header on every request:

- **Local navigation** — the `Referer` is a relative URL, or an absolute URL whose host matches the current request's
  host — the referring page is **appended to the trail** as the newest breadcrumb.
- **Non-local navigation** — the `Referer` is missing, unparseable, or points at a *different* host (the user arrived
  from search, a link on another site, a fresh tab, etc.) — the trail is **reset**. This is treated as the start of a
  new visit.

The trail itself lives in the scs session, so it survives across requests for that user/browser but is scoped
per-session like anything else in scs.

## Install

```shell
go get github.com/example/pagebreadcrumbs
go get github.com/alexedwards/scs/v2
go get github.com/go-chi/chi/v5
```

(Adjust the module path to wherever you actually host this package; update `go.mod`'s `module` line and the import in
`example/main.go` to match.)

## Usage

```go
sessionManager := scs.New()
sessionManager.Lifetime = 24 * time.Hour

trail := breadcrumbs.New(sessionManager)

r := chi.NewRouter()
r.Use(sessionManager.LoadAndSave) // must come first
r.Use(trail.Middleware)           // reads/writes the session

r.Get("/page/{name}", func(w http.ResponseWriter, req *http.Request) {
    ctx := req.Context()

    if prev, ok := trail.Previous(ctx); ok {
        fmt.Fprintf(w, "came from: %s\n", prev)
    }

    if first, ok := trail.At(ctx, 0); ok {
        fmt.Fprintf(w, "first page in this trail: %s\n", first)
    }

    fmt.Fprintf(w, "full trail: %v\n", trail.All(ctx))
})
```

See `example/main.go` for a complete, runnable version.

## API

- `New(session *scs.SessionManager) *Manager` — construct a Manager.
- `(*Manager).Middleware(next http.Handler) http.Handler` — chi-compatible
  middleware; mount after `sessionManager.LoadAndSave`.
- `(*Manager).Previous(ctx) (string, bool)` — the most recently recorded
  breadcrumb (the page the user came from to reach the current request).
- `(*Manager).At(ctx, index int) (string, bool)` — the breadcrumb at
  `index`, where `0` is the oldest since the trail was last reset and
  `Len(ctx)-1` is the most recent (same as `Previous`).
- `(*Manager).All(ctx) []string` — the full trail, oldest first.
- `(*Manager).Len(ctx) int` — number of breadcrumbs currently stored.
- `(*Manager).Reset(ctx)` — manually clear the trail (e.g. on logout).

## Notes / things to configure for your setup

- **Host detection**: `Manager.HostFunc` decides what counts as "local". It defaults to `r.Host`, falling back to the
  `X-Forwarded-Host` header when present. If you're behind a proxy/load balancer that uses a different header (or you
  need to allow multiple hostnames as "local"), override `HostFunc`:

  ```go
  trail.HostFunc = func(r *http.Request) string {
      return "www.example.com" // or inspect r however you need
  }
  ```

- **What gets stored**: the raw `Referer` header value is stored as-is (whatever the browser sent — relative or
  absolute). If you'd rather normalize to paths only, resolve/trim in `record` before calling `push`, or post-process
  the values returned by `All`/`At`/`Previous`.
- **gob registration**: scs's default codec uses `encoding/gob`, which needs concrete types registered for values stored
  as `interface{}`. This package registers `[]string` in its `init()`, so no extra setup is needed unless you swap in a
  custom scs codec (e.g. `scs`'s JSON codec), in which case the registration is simply unnecessary but harmless.

## Tests

```shell
go test ./...
```
