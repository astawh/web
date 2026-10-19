-- +goose Up
CREATE TABLE events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL CHECK(length(trim(title)) BETWEEN 1 AND 200),
    description TEXT NOT NULL CHECK(length(description)<=50000),
    campus TEXT NOT NULL DEFAULT '' CHECK(campus IN ('','gelsenkirchen','recklinghausen','bocholt')),
    location TEXT NOT NULL DEFAULT '' CHECK(length(location)<=500),
    link TEXT NOT NULL DEFAULT '' CHECK(length(link)<=2048),
    status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','published','cancelled')),
    all_day INTEGER NOT NULL DEFAULT 0 CHECK(all_day IN (0,1)),
    starts_at TEXT CHECK(starts_at IS NULL OR (length(starts_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',starts_at,'+0 seconds') IS starts_at)),
    ends_at TEXT CHECK(ends_at IS NULL OR (length(ends_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',ends_at,'+0 seconds') IS ends_at)),
    created_at TEXT NOT NULL CHECK(length(created_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',created_at,'+0 seconds') IS created_at),
    updated_at TEXT NOT NULL CHECK(length(updated_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',updated_at,'+0 seconds') IS updated_at),
    deleted_at TEXT CHECK(deleted_at IS NULL OR (length(deleted_at)=20 AND strftime('%Y-%m-%dT%H:%M:%SZ',deleted_at,'+0 seconds') IS deleted_at)),
    CHECK(ends_at IS NULL OR (starts_at IS NOT NULL AND ends_at>starts_at)),
    CHECK(all_day=0 OR (starts_at IS NOT NULL AND ends_at IS NOT NULL))
) STRICT;
CREATE INDEX events_public_schedule ON events(status,starts_at,id) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE events;
