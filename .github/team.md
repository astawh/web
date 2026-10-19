# Team positions and office hours

The Go/SQL model follows the structure of the current
[team page](https://www.asta-wh.de/team/): members appear under organizational
areas, with a position, shared contact email and individual office hours.

## Structure

- `team_positions` belongs to an `organization_units` record and supplies a title,
  public contact email, display order and active state.
- `team_assignments` associates an existing staff `users` record with a position.
  One user may hold multiple positions; one position may have multiple users.
  Duplicate user/position pairs are rejected. Publication is explicit and defaults
  to off.
- `office_hours` belongs to an assignment. Multiple arrangements can describe
  different days, campuses or appointment options. Removing an assignment also
  removes its office hours.

There is no separate people/profile record. Names come from the assigned user;
the public query returns the position's contact email, never the account email.
Assignments do not create access roles or organizational memberships. Positions
can be deactivated while retaining their assignments.

## Arrangements

| Kind | Meaning |
| --- | --- |
| `weekly` | ISO weekday (`1` Monday through `7` Sunday), start and end local time |
| `appointment` | By appointment (`n. V.`), without a fixed weekday or interval |
| `external` | Reference to another schedule, e.g. `/shop` |

Weekly intervals use canonical `HH:MM` values, end after start on the same day,
and cannot overlap within an assignment. Adjacent intervals are allowed.
Nonweekly arrangements cannot carry weekly fields. External schedule references
accept HTTP(S) URLs or root-relative paths; other arrangements have no schedule URL.

Every arrangement can specify:

- Campus: `gelsenkirchen`, `recklinghausen` or `bocholt`.
- Room and meeting mode: `in_person`, `remote` or `hybrid`. Physical/hybrid
  arrangements require a campus; remote arrangements can omit it.
- `LectureOnly`: whether the advertised arrangement applies only during the
  lecture period.
- Notes: for example, “via Teams” or appointments during the lecture-free period.

`LectureOnly` is schedule metadata. This slice does not maintain academic term
calendars or automatically calculate holiday closures. The existing page's rule
can be represented by lecture-only weekly hours plus an appointment arrangement
and explanatory notes.

## Local recurrence versus UTC timestamps

A weekly Wednesday 13:00 slot is a wall-clock recurrence, not an instant. Its
timezone is explicitly `Europe/Berlin`, enforced in SQL. It stays at 13:00 locally
through daylight-saving changes. Do not turn it into a fixed UTC weekly time.

Position and assignment `created_at`/`updated_at` values use the same constrained
UTC RFC3339 representation as the rest of the database. If concrete appointments
are added later, their actual instants must also use UTC RFC3339.

## Go services and authorization

`model/team.New(db)` exposes:

- `SavePosition`: create or update the position, including its active state.
- `Positions`: administration catalogue, including inactive positions.
- `SaveAssignment`: create or update a member assignment and replace all its
  arrangements in one transaction.
- `Assignments`: administration listing with office hours.
- `RemoveAssignment`: remove the assignment and its arrangements atomically.
- `PublicMembers`: published team data for future SSR rendering, grouped by
  organizational area and position with deterministic display ordering.

Administration requires `organization.manage`; mutations check the actor inside
their transaction and write an audit event. No access is implied by being the
assigned member. Assignment targets must be enabled existing users.

The public query filters unpublished assignments, inactive positions and disabled
accounts. Reads use a consistent transaction for assignments and their hours.
These services are ready for later SSR/MVC integration.
