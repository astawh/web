-- +goose Up
-- Organizational positions are public responsibilities, never access roles.
CREATE TABLE team_positions (
    id TEXT PRIMARY KEY,
    organization_unit_id TEXT NOT NULL REFERENCES organization_units(id),
    title TEXT NOT NULL CHECK(length(trim(title)) BETWEEN 1 AND 200),
    contact_email TEXT NOT NULL DEFAULT '' CHECK(length(contact_email)<=254),
    sort_order INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL CHECK(active IN (0,1)),
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at),
    updated_at TEXT NOT NULL CHECK(length(updated_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',updated_at,'+0 seconds') IS updated_at)
) STRICT;
CREATE INDEX team_positions_unit ON team_positions(organization_unit_id);

CREATE TABLE team_assignments (
    id TEXT PRIMARY KEY,
    position_id TEXT NOT NULL REFERENCES team_positions(id),
    user_id TEXT NOT NULL REFERENCES users(id),
    published INTEGER NOT NULL DEFAULT 0 CHECK(published IN (0,1)),
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at),
    updated_at TEXT NOT NULL CHECK(length(updated_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',updated_at,'+0 seconds') IS updated_at),
    UNIQUE(position_id,user_id)
) STRICT;
CREATE INDEX team_assignments_user ON team_assignments(user_id);

-- Weekly wall-clock values are deliberately not UTC timestamps. Europe/Berlin
-- preserves the advertised local hour across daylight-saving transitions.
CREATE TABLE office_hours (
    id TEXT PRIMARY KEY,
    assignment_id TEXT NOT NULL REFERENCES team_assignments(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('weekly','appointment','external')),
    weekday INTEGER CHECK(weekday BETWEEN 1 AND 7), -- ISO Monday=1
    start_time TEXT CHECK(length(start_time)=5 AND start_time GLOB '[0-2][0-9]:[0-5][0-9]' AND start_time<'24:00'),
    end_time TEXT CHECK(length(end_time)=5 AND end_time GLOB '[0-2][0-9]:[0-5][0-9]' AND end_time<'24:00'),
    timezone TEXT NOT NULL DEFAULT 'Europe/Berlin' CHECK(timezone='Europe/Berlin'),
    campus TEXT NOT NULL DEFAULT '' CHECK(campus IN ('','gelsenkirchen','recklinghausen','bocholt')),
    room TEXT NOT NULL DEFAULT '' CHECK(length(room)<=100),
    meeting_mode TEXT NOT NULL CHECK(meeting_mode IN ('in_person','remote','hybrid')),
    schedule_url TEXT NOT NULL DEFAULT '' CHECK(length(schedule_url)<=2048),
    lecture_only INTEGER NOT NULL CHECK(lecture_only IN (0,1)),
    notes TEXT NOT NULL DEFAULT '' CHECK(length(notes)<=2000),
    sort_order INTEGER NOT NULL DEFAULT 0,
    CHECK((kind='weekly' AND weekday IS NOT NULL AND start_time IS NOT NULL AND end_time IS NOT NULL AND start_time<end_time)
       OR (kind!='weekly' AND weekday IS NULL AND start_time IS NULL AND end_time IS NULL)),
    CHECK((kind='external' AND length(schedule_url)>0) OR (kind!='external' AND schedule_url='')),
    CHECK(meeting_mode='remote' OR campus!='')
) STRICT;
CREATE INDEX office_hours_assignment ON office_hours(assignment_id);

-- +goose Down
DROP TABLE office_hours;
DROP TABLE team_assignments;
DROP TABLE team_positions;
