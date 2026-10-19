# web

The website of the AStA at Westfälische Hochschule.

```text
           site
         /  |  \
        ▼   ▼   ▼
     model view storage
```

## Getting started

Requires [Nix](https://nixos.org/download/) with flakes and `nix-command` enabled.

```sh
nix develop -c go run .
```

Open <http://127.0.0.1:8090/>. Alternatively, run `docker compose up --build`.

See the [contributing guide](.github/CONTRIBUTING.md) for development and checks,
and the [architecture guide](.github/architecture.md) for the repository layout.

The [backend guide](.github/backend.md) describes invitation-only staff access,
roles, organizational memberships, database setup and first-administrator creation.

The [frontend guide](.github/frontend.md) covers the GOV.UK-based public layout,
navigation and reproducible browser assets.

SPDX-License-Identifier: MPL-2.0
