# Public layout

The shared SSR shell consists of `view/templates/layout.html`, `header.html`
and `footer.html`. `landing.html` provides the initial public entry page.
`navigation.html` contains the non-JavaScript topic index and search results.
Page data in `view` remains independent of domain services.
The landing content is intentionally empty while the shared layout is developed.

## GOV.UK components

The GOV.UK-style header reproduces the initial `index.html` prototype: 60px
desktop / 55px mobile bar, 19px desktop menu links,
underlined group headings and the compact two-column services grid. Local SCSS
preserves those measured dimensions. The blue header and full-width pale-blue
panels use AStA branding, German labels and AStA destinations. Menu/search
toggling and keyboard interaction use GOV.UK's super-navigation JavaScript.

Pinned dependencies in `flake.nix`:

- GOV.UK Frontend 6.6.0 for typography, grid, footer and base components.
- GOV.UK Publishing Components 70.2.1 for super-navigation JavaScript.

Assets are built with Dart Sass through Nix and served locally. The font is
Arial/system sans-serif. Upstream licences are included in `view/assets`.

`view/assets/logo.svg` is the original blue AStA WH logo from
`https://www.asta-wh.de/wp-content/uploads/2016/08/Logo_blau.svg`, preserved as
a local asset. The header uses it at 88×44px (80×40px on mobile), displayed
white against the blue bar. Keyboard focus and printing display it in black.
The footer uses the original blue logo at 160×80px. Both logos link to the
homepage and have accessible names.

## Editing assets

Edit `view/styles/*.scss` and `view/site.js`, then regenerate:

```sh
nix run .#vendor
```

Commit the generated `view/assets` alongside source changes. This makes plain
`go run .` and the Docker build usable without Node or Sass at runtime. Nix builds
regenerate embedded assets, and `nix flake check` verifies the committed assets
match their sources and pinned dependencies.

## Navigation and search

`site/navigation.go` supplies the compact header menu and complete topic
catalogue. Links point to the
current AStA website until their local SSR replacements are implemented.

With JavaScript, menu and search buttons open mutually exclusive panels. Escape
closes a panel and returns focus to its toggle. Without JavaScript, the header
links lead to `/navigation` and `/suche`. Both routes are ordinary server-rendered
GET pages. `/suche?q=...` searches topic titles, descriptions and group names;
it does not yet search database content. No third-party requests are made when
loading the layout.
