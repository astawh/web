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

`/formulare` uses the GOV.UK browse-page structure: a pale-blue (`#d2e2f1`)
title/summary area with dark text and a homepage breadcrumb, frequently used
action links followed by a 4px blue divider, and a ruled list of documents
with descriptions and chevrons. Documents are grouped under Sportkeller and
Veranstaltungen. Links point to the existing AStA PDFs, including the current
registration document linked from the fitness page. The catalogue is defined
in `site/forms.go`; presentation stays in `view/templates/forms.html`.

`/asta/so-funktioniert-der-asta` establishes the long-form content-page pattern
from GOV.UK's “How government works”: white background, hierarchical breadcrumbs,
48px desktop title, 24px lead paragraph, full-width divider, dashed contents
links and a two-thirds-width reading column. Sections use 36px desktop headings
and 60px spacing (40px mobile). `content.html` contains the shared header and
contents partials; `asta.html` contains the editorial body. The route and
presentation data live in `site/content.go`. No JavaScript is required to read
the page or navigate its contents.

The breadcrumb is “Startseite → Hochschulpolitik → Gremien”. Its parent pages
at `/hochschulpolitik` and `/hochschulpolitik/gremien` use the browse layout
in `browse.html`, with linked topics and descriptions. The content page keeps
its existing URL; the breadcrumb expresses the editorial hierarchy.

`/asta/team` uses the role-first summary-list structure of `fsi-wh.de/team`.
Sections cover Vorstand, Referate and Verwaltung. Each responsibility has its
members' names, campus, room and individual office hours, followed by one shared
email address. On mobile, the responsibility stacks above its members. Weekly,
appointment-only, Teams and external shop schedules retain their distinctions;
all times are local German time. The page uses no portraits or biographies.

`site/team.go` holds the initial editorial snapshot of the 18 people published
at `https://www.asta-wh.de/team/` on 2026-10-19. It does not seed database users
or permissions; connecting the directory to managed team records is a separate
integration. `view/templates/team.html` renders presentation-only team data.

Editorial sources checked on 2026-10-19:

- [AStA team and office hours](https://www.asta-wh.de/team/): responsibilities,
  departments, finance contact, office hours and how to join.
- [About the StuPa](https://www.stupa-wh.de/das-stupa): elections, oversight,
  budget and fundamental decisions.
- [StuPa vacancies](https://www.stupa-wh.de/ausschreibungen): recruitment route;
  dated vacancy details and deadlines are not copied into the content page.
- [WH semester contribution](https://www.w-hs.de/rueckmeldung/): funding context;
  changing amounts remain on the university website.

The shared footer follows GOV.UK's footer navigation: a
two-thirds/two-column “Angebote und Informationen” group and a one-third
“Über den AStA” group. Curated destinations live in `site/navigation.go`.
The official footer component supplies the pale background, blue top border,
underlined headings, responsive columns and separator. Twelve service/topic
 links help students find support and practical offers; institutional links
cover people, news, decisions, transparency, representation and participation.
Support/legal links sit below one divider, with the AStA
logo at the bottom right. The footer contains no campus address blocks.
Spacing matches the live GOV.UK footer: 40px below each navigation list on
desktop and mobile, 30px below heading rules, and 50px (30px mobile) between
the separator and support row. The logo replaces GOV.UK's copyright emblem.

The links to `/asta/protokolle`, `/asta/transparenz`, `/asta/mitmachen`, `/hilfe`
and `/barrierefreiheit` are provisional navigation destinations; their pages
are not implemented yet.

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
