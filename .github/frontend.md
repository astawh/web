# Public layout

The shared SSR shell consists of `view/templates/layout.html`, `header.html`
and `footer.html`. `landing.html` provides the initial public entry page.
`navigation.html` contains the non-JavaScript topic index and search results.
Page data in `view` remains independent of domain services.
The landing page has a blue introduction/search area directly beneath the logo
row, following the public GOV.UK homepage. Its heading and always-visible search
form share the header's colour and content alignment.

Below it, “Häufig gesucht” follows GOV.UK's popular-links section: three desktop
columns, a stacked mobile list, bold 19px links and decorative 40px circular
arrow SVGs from the GOV.UK action-link component (covered by the vendored
components licence). The shortcuts are Semesterticket, Sportkeller-Anmeldung
and Beratung. The registration shortcut currently opens the existing AStA PDF.

## GOV.UK components

The GOV.UK-style header uses the initial `index.html` prototype's 19px desktop
menu links, underlined group headings and compact two-column services grid.
The masthead is 112px high on desktop and 88px on mobile to accommodate the
2:1 AStA logo. Desktop controls use 19px type; mobile controls use 16px.
The blue header and full-width pale-blue
panels use AStA branding, German labels and AStA destinations. Menu/search
toggling and keyboard interaction use GOV.UK's super-navigation JavaScript.

Pinned dependencies in `flake.nix`:

- GOV.UK Frontend 6.6.0 for typography, grid, footer and base components.
- GOV.UK Publishing Components 70.2.1 for super-navigation JavaScript.

Assets are built with Dart Sass through Nix and served locally. The font is
Arial/system sans-serif. Upstream licences are included in `view/assets`.

`view/assets/logo.svg` is the original blue AStA WH logo from
`https://www.asta-wh.de/wp-content/uploads/2016/08/Logo_blau.svg`, preserved as
a local asset. The header uses it at 176×88px (136×68px on mobile), displayed
white against the blue bar. Keyboard focus and printing display it in black.
The footer uses the original blue logo at 160×80px. Both logos link to the
homepage and have accessible names.

Sizing follows the artwork's aspect ratio: the
[GOV.UK header](https://design-system.service.gov.uk/components/header/) uses a
162×30px horizontal wordmark, while the original AStA artwork is 200×100 and
includes a small caption. The
[generic header guidance](https://design-system.service.gov.uk/components/generic-header/)
supports using an organisation's own logo. A taller masthead keeps the AStA
artwork legible; its grid keeps the menu/search controls aligned and both
dropdowns full-width beneath it.

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
