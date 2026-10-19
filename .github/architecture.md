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
`/assets/`. `model/organization` supplies the site identity. `model/access` and
`model/users` implement private staff access, organizational responsibilities,
invitations and sessions. `model/team` owns organizational positions, user
assignments and office-hour arrangements. SQLite migrations are embedded in
`internal/database`; TLS SMTP delivery is provided by `internal/email`.
`model/events` manages event content, publication and scheduling.

There is no REST API. The public shell, topic index and topic search are
server-rendered. Staff controllers and administration screens will be added
later. See the [backend guide](backend.md) for the implemented model and
initialization commands.

As further editable content is introduced, add domain packages such as
`model/services`. Keep private user accounts separate from
organizational positions; public profiles
are not required. Persistence uses parameterized `database/sql` queries in domain
services and versioned Goose migrations. Add disk/S3 implementations
under `storage` when uploaded files are introduced.

The public layout uses pinned GOV.UK Frontend and GOV.UK Publishing Components
assets built through Nix, with handwritten styles under `view/styles` and browser
code under `view`. Generated assets are committed for plain Go/Docker builds and
checked for reproducibility. See the [frontend guide](frontend.md).
