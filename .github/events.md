# Events backend

`model/events.New(db)` provides event management for future SSR controllers.
Records contain a title, plain-text description, optional campus/location and
HTTP(S) information or registration link. Campus values match the three campuses
used by the team model; an empty campus can represent cross-campus or online
events. Text is stored as text, not trusted HTML.

## Publication and authorization

- `draft` is the default and is only visible to content managers.
- `published` is public.
- `cancelled` remains publicly addressable as a cancellation announcement, but is
  excluded from upcoming/promotional listings.

`Save`, `Lookup`, `List` and `Delete` require `content.manage`. An administrator
needs the explicit content-editor role to manage events, just like any other
user. Organizational memberships and team positions do not grant this permission.
Disabled accounts cannot manage events.

All mutations check permissions and write an audit event inside their transaction.
`Save` creates when its ID argument is empty and updates otherwise. It accepts
the complete editable state; omitting status means draft, not preservation of the
old publication state. `Delete` soft-deletes a record. Deleted events cannot be
read, edited or publicly listed through these services; their audit trail remains.

## Scheduling

`Draft.Schedule` supports:

| Kind | Input |
| --- | --- |
| `unscheduled` (default) | No date/time values; suitable for an announcement with a date to follow |
| `timed` | `StartsAt` and optional `EndsAt`, as RFC3339 instants with explicit zones |
| `all_day` | `StartDate` and optional exclusive `EndDate`, as `YYYY-MM-DD` |

Timed inputs are converted to UTC and stored at whole-second precision, ending
in `Z`. An end must remain strictly after its start at that stored precision.
Mixing date-only fields and timestamp fields is rejected.

All-day dates are interpreted in `Europe/Berlin`. An omitted end means the next
calendar day. The stored interval runs from local midnight at the start through
local midnight at the exclusive end, converted to UTC. An all-day event can
therefore span 23 or 25 hours during daylight-saving changes. Embedded Go timezone
data makes this work in the minimal container image. The `AllDay` flag preserves
the distinction from a timed event; use the Berlin calendar dates for all-day
display rather than rendering the stored UTC dates directly.

All stored scheduling, creation, update and deletion timestamps are constrained
UTC RFC3339 text. Unset scheduling values use SQL NULL.

## Queries

- `Lookup(ctx, actor, id)`: one undeleted record, including drafts.
- `List(ctx, actor, page)`: administration listing, including drafts.
- `PublicLookup(ctx, id)`: one published/cancelled record; draft/deleted/missing
  IDs all return `ErrNotFound`.
- `PublicList(ctx, now, period, page)`: public timeline for `all`, `upcoming` or
  `past`. The caller supplies the comparison instant.

Ongoing intervals count as upcoming until their exclusive end. A timed event
without an end is upcoming through its start instant and past afterward.
Unscheduled announcements appear last and never count as past. Upcoming excludes
cancelled events; all/past retain them. Past results are newest-first; other lists
are earliest-first, with stable ID tie-breaks.

Pagination defaults to 20 results and accepts 1–100 per page with a nonnegative
offset. Invalid input uses `access.ErrInvalid`; missing/deleted events use
`events.ErrNotFound`; permission failures use `access.ErrForbidden`.
