# Access and invitation backend

## Scope and integration

The access foundation is implemented as Go domain services and embedded SQLite
migrations. There are no REST endpoints and no administration frontend yet.
Future SSR controllers in `site` will call these services, map their results into
presentation-only `view` types, and render full pages or HTMX HTML fragments.

No public registration, student accounts, people table or public profiles are
created. Form records currently define responsibility boundaries only; form
definitions, student applications and their email-link access are later domains.

## Database

```text
users ── user_roles ── roles ── role_permissions ── permissions
  │
  ├── sessions
  └── user_memberships ── organization_units ── form_responsibilities ── forms

invitations
  ├── invitation_roles ── roles
  ├── invitation_memberships ── organization_units
  └── mail_outbox

access_events
```

`users` contains private staff account information. Organizational areas use
`board`, `department` or `project`; there is no hierarchy or inherited access.
Memberships grant `view` or `manage` access within an explicit area.

Every database is migrated at startup using Goose. Foreign keys are enabled,
tables are STRICT, and changes to grants or invitation acceptance are committed
transactionally. Timestamps are UTC Unix seconds. New database files use mode
0600 and their parent directory is created with mode 0700.

### Functional roles

| Role key | Permission |
| --- | --- |
| `administrator` | `users.manage`, `organization.manage` |
| `content_editor` | `content.manage` |
| `organization_editor` | `organization.manage` |
| `application_handler` | `applications.process` |

A user may have multiple roles. The last enabled user with `users.manage` cannot
be disabled or have that permission removed. Access changes are recorded in
`access_events`; disabling an account deletes its sessions.

Application access requires **both** `applications.process` and an explicit
membership in an area responsible for the form. `manage` is required for writing.
An administrator has no implicit application access. Administrators can explicitly
assign themselves access, which produces the same access-change event as any
other grant.

## Go entry points

`model/access.Service`:

- `CreateUnit`, `Units`: organizational areas.
- `Roles`: role/permission catalogue for an invitation or access-edit screen.
- `Users`, `UserGrants`, `SetUserAccess`: staff administration.
- `CreateForm`, `Forms`, `SetFormResponsibilities`: form responsibility metadata.
- `CanAccessForm`: authorization for future application operations.

`model/users.Service`:

- `Bootstrap`: first administrator, only while there are no users.
- `Invite`, `Invitations`, `ResendInvitation`, `RevokeInvitation`.
- `AcceptInvitation`: create the user and transfer precisely the offered grants.
- `Login`, `Authenticate`, `Logout`, `Grants`: staff sessions.
- `DeliverInvitations`: leased outbox delivery and retry backoff.
- `Cleanup`: expired sessions and obsolete invitation mail.

Service methods return typed sentinel errors for forbidden, invalid, conflicting
or unavailable operations. Authorization checks for mutations happen inside
their transactions. SSR controllers must supply the actor from the authenticated
session, never from submitted user IDs.

## Invitation lifecycle

1. An authorized administrator supplies name, email, roles and memberships.
2. The invitation and email queue entry commit together; no user exists yet.
3. A seven-day, single-use token is sent to the invited email address.
4. Acceptance rechecks expiry, revocation and the inviter's current permission.
5. User creation, grants, acceptance and queue deletion commit together.

Resending rotates the token, renews expiry, and makes the resending administrator
responsible for the offered grants. Revocation removes queued mail. An accepted
or revoked invitation cannot be resent. Existing users must be managed through
their account rather than invited again.

Invitation and session token fingerprints are SHA-256 hashes of random 256-bit
credentials. Pending email necessarily contains the original invitation token;
the restricted outbox deletes it after delivery, acceptance, revocation or expiry.
Delivery is at-least-once: a crash immediately after SMTP acceptance may resend
the same link. Delivery uses leases so concurrent workers do not normally send
the same queue entry.

Passwords use bcrypt cost 12, with at least 12 Unicode characters and at most
72 UTF-8 bytes. Sessions expire after 12 hours and recheck account disablement
on every authentication.

## First administrator

Create a private file containing a password, with permissions `0600`, then run:

```sh
nix develop -c go run . \
  -database data/sqlite.db \
  -bootstrap-admin administrator@asta-wh.de \
  -bootstrap-name Administrator \
  -bootstrap-password-file /absolute/path/to/private-password-file
```

The command exits after creation and cannot create another administrator once
any user exists. The password is not passed as a command-line argument or logged.
Remove the temporary password file after setup. No accounts are automatically
created on normal startup.

To apply migrations without starting the server:

```sh
nix develop -c go run . -database data/sqlite.db -migrate-only
```

## Configuration

| Variable | Default / meaning |
| --- | --- |
| `SITE_NAME` | `AStA WH` |
| `DATABASE_PATH` | `data/sqlite.db`; overridden by `-database` |
| `PUBLIC_URL` | `http://127.0.0.1:8090`; absolute origin for future SSR links |
| `SMTP_ADDRESS` | Optional SMTP host and port; absent means mail stays queued |
| `SMTP_FROM` | Required sender address when SMTP is enabled |
| `SMTP_USERNAME` | Optional authentication username |
| `SMTP_PASSWORD` | Required with the username |
| `SMTP_MODE` | `starttls` (default) or `tls` for implicit TLS |

SMTP verifies server certificates and requires TLS. The background worker runs
at startup and every 30 seconds. SMTP acceptance is transport handoff, not proof
of inbox delivery. The invitation email reserves `/einladung?token=...` for the
future SSR acceptance controller; enable real invitations once that screen is
implemented. The Go acceptance service is already usable and integration-tested.

## SSR/HTMX boundary

Future controllers must add HTTP concerns around these services: CSRF validation,
same-origin checks, bounded form bodies, login/invitation rate limiting and secure
HttpOnly session cookies. Tokens and passwords must not be rendered into logs.
The database and domain layer deliberately do not depend on browser behavior.
