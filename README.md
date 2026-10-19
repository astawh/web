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

SPDX-License-Identifier: MPL-2.0
