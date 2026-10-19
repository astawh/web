-- +goose Up
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 200),
    email TEXT NOT NULL UNIQUE CHECK(email=lower(trim(email))),
    password_hash TEXT NOT NULL,
    verified_at TEXT NOT NULL CHECK(length(verified_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',verified_at,'+0 seconds') IS verified_at),
    disabled_at TEXT CHECK(disabled_at IS NULL OR (length(disabled_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',disabled_at,'+0 seconds') IS disabled_at)),
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at),
    updated_at TEXT NOT NULL CHECK(length(updated_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',updated_at,'+0 seconds') IS updated_at)
) STRICT;

CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
) STRICT;
CREATE TABLE permissions (
    id TEXT PRIMARY KEY,
    description TEXT NOT NULL
) STRICT;
CREATE TABLE user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id),
    PRIMARY KEY(user_id,role_id)
) STRICT;
CREATE TABLE role_permissions (
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id TEXT NOT NULL REFERENCES permissions(id),
    PRIMARY KEY(role_id,permission_id)
) STRICT;

INSERT INTO permissions VALUES
 ('users.manage','Invite users and manage their access'),
 ('organization.manage','Manage organizational units and form responsibilities'),
 ('content.manage','Edit public content'),
 ('applications.process','Process applications within assigned organizational areas');
INSERT INTO roles VALUES
 ('administrator','Administrator'),
 ('content_editor','Content editor'),
 ('organization_editor','Organization editor'),
 ('application_handler','Application handler');
INSERT INTO role_permissions VALUES
 ('administrator','users.manage'),('administrator','organization.manage'),
 ('content_editor','content.manage'),
 ('organization_editor','organization.manage'),
 ('application_handler','applications.process');

CREATE TABLE organization_units (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 200),
    kind TEXT NOT NULL CHECK(kind IN ('board','department','project')),
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at)
) STRICT;
CREATE TABLE user_memberships (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_unit_id TEXT NOT NULL REFERENCES organization_units(id),
    access_level TEXT NOT NULL CHECK(access_level IN ('view','manage')),
    PRIMARY KEY(user_id,organization_unit_id)
) STRICT;

-- Form metadata establishes responsibility boundaries; application data will
-- be introduced with the form-processing feature.
CREATE TABLE forms (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE CHECK(length(trim(key))>0),
    title TEXT NOT NULL CHECK(length(trim(title))>0),
    active INTEGER NOT NULL DEFAULT 1 CHECK(active IN (0,1))
) STRICT;
CREATE TABLE form_responsibilities (
    form_id TEXT NOT NULL REFERENCES forms(id) ON DELETE CASCADE,
    organization_unit_id TEXT NOT NULL REFERENCES organization_units(id),
    PRIMARY KEY(form_id,organization_unit_id)
) STRICT;

CREATE TABLE invitations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 200),
    email TEXT NOT NULL CHECK(email=lower(trim(email))),
    token_hash TEXT NOT NULL UNIQUE,
    invited_by TEXT NOT NULL REFERENCES users(id),
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at),
    expires_at TEXT NOT NULL CHECK(length(expires_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',expires_at,'+0 seconds') IS expires_at AND expires_at>created_at),
    accepted_at TEXT CHECK(accepted_at IS NULL OR (length(accepted_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',accepted_at,'+0 seconds') IS accepted_at)),
    revoked_at TEXT CHECK(revoked_at IS NULL OR (length(revoked_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',revoked_at,'+0 seconds') IS revoked_at)),
    CHECK(accepted_at IS NULL OR revoked_at IS NULL)
) STRICT;
CREATE UNIQUE INDEX invitations_pending_email ON invitations(email)
 WHERE accepted_at IS NULL AND revoked_at IS NULL;
CREATE TABLE invitation_roles (
    invitation_id TEXT NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id),
    PRIMARY KEY(invitation_id,role_id)
) STRICT;
CREATE TABLE invitation_memberships (
    invitation_id TEXT NOT NULL REFERENCES invitations(id) ON DELETE CASCADE,
    organization_unit_id TEXT NOT NULL REFERENCES organization_units(id),
    access_level TEXT NOT NULL CHECK(access_level IN ('view','manage')),
    PRIMARY KEY(invitation_id,organization_unit_id)
) STRICT;

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at),
    expires_at TEXT NOT NULL CHECK(length(expires_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',expires_at,'+0 seconds') IS expires_at AND expires_at>created_at)
) STRICT;
CREATE INDEX sessions_user ON sessions(user_id);
CREATE INDEX sessions_expiry ON sessions(expires_at);

CREATE TABLE mail_outbox (
    id TEXT PRIMARY KEY,
    invitation_id TEXT NOT NULL UNIQUE REFERENCES invitations(id) ON DELETE CASCADE,
    recipient TEXT NOT NULL,
    token TEXT NOT NULL,
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at),
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TEXT NOT NULL CHECK(length(next_attempt_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',next_attempt_at,'+0 seconds') IS next_attempt_at),
    lease_token TEXT,
    lease_until TEXT CHECK(lease_until IS NULL OR (length(lease_until)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',lease_until,'+0 seconds') IS lease_until))
) STRICT;
CREATE INDEX mail_outbox_due ON mail_outbox(next_attempt_at);

CREATE TABLE access_events (
    id INTEGER PRIMARY KEY,
    actor_id TEXT REFERENCES users(id),
    subject_id TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at)
) STRICT;

-- +goose Down
DROP TABLE access_events;
DROP TABLE mail_outbox;
DROP TABLE sessions;
DROP TABLE invitation_memberships;
DROP TABLE invitation_roles;
DROP TABLE invitations;
DROP TABLE form_responsibilities;
DROP TABLE forms;
DROP TABLE user_memberships;
DROP TABLE organization_units;
DROP TABLE role_permissions;
DROP TABLE user_roles;
DROP TABLE permissions;
DROP TABLE roles;
DROP TABLE users;
