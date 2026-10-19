# Contributing

## Development

```sh
nix develop
go run .
```

The application listens on `127.0.0.1:8090`. Use `-addr` to change it.
`SITE_NAME` overrides the default public name, `AStA WH`.

Templates and static assets are embedded in the executable. Restart `go run .`
after changing them. The initial application uses Go's standard library only.

## Checks

```sh
nix develop -c go test -race ./...
nix develop -c go vet ./...
nix develop -c actionlint
nix flake check
```

Format Go with `gofmt`. CI is temporarily disabled: workflow definitions are
kept in `.github/workflows-disabled/` until explicitly re-enabled. Run the checks
locally before committing.
Go's version in `go.mod` must match the Docker builder; Nix and CI derive their
Go version from `go.mod`.

## Commits and pull requests

Use Conventional Commits: `type(scope): description`, with an optional scope
and `!` breaking-change marker. Keep descriptions lowercase and at most
50 characters, excluding the prefix. Supported punctuation is `. / @ _ + -`.
Scopes are lowercase alphanumeric.

Examples: `feat: add contact page`, `fix(site): correct response status`.
When squash merging, use the PR title as the commit subject.
