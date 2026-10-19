# Architecture

This project follows the server-rendered Go structure of `fsiwh/web`.

## Boundaries

- `main.go` handles command-line flags, HTTP server lifetime and graceful shutdown.
- `application.go` constructs the services and connects them to the HTTP layer.
- `internal/config` reads process settings at startup.
- `model/<domain>` owns domain data and application rules. It does not import
  `site` or `view`.
- `site` owns routes, HTTP semantics and mapping domain data to view data.
- `view` owns presentation-only page types, escaped HTML templates and embedded
  browser assets. It does not import domain, HTTP-handler or storage packages.
- `storage` is the boundary for future file/object storage implementations.
  It must not depend on the domain or presentation layers.

## Initial scope

The application serves a minimal homepage, `/healthz` and embedded assets under
`/assets/`. `model/organization` supplies the site identity. No external Go modules
or database are required at this stage.

As editable content is introduced, add domain packages such as `model/posts`,
`model/events`, `model/services` and `model/members`. Follow the reference project's
`internal/database` approach for SQLite connection setup, embedded migrations and
generated SQLC queries when persistence is needed. Add disk/S3 implementations
under `storage` when uploaded files are introduced.

The production frontend should build pinned GOV.UK Frontend assets through Nix,
with handwritten styles under `view/styles` and browser code under `view`.
The bootstrap stylesheet is only for the minimal application shell.
