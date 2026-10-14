-- Schema v1: the content model from the concept (posts, contexts, tags,
-- pages, media, slug history), the admin's users and sessions, and settings.
-- db/SCHEMA.md maps each concept entity to its table.

-- +goose Up

CREATE TYPE lang AS ENUM ('ru', 'en');

-- Scheduled is not a status: a scheduled post is 'published' with a future
-- publish_at, so public queries filter publish_at <= now() and no job runs.
CREATE TYPE post_status AS ENUM ('draft', 'published', 'unpublished');

-- Media

CREATE TABLE media (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    storage_key text        NOT NULL UNIQUE,
    filename    text        NOT NULL,
    mime_type   text        NOT NULL,
    size_bytes  bigint      NOT NULL CHECK (size_bytes >= 0),
    width       integer     CHECK (width > 0),
    height      integer     CHECK (height > 0),
    private     boolean     NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CHECK ((width IS NULL) = (height IS NULL))
);

CREATE TABLE media_variants (
    media_id    bigint  NOT NULL REFERENCES media (id) ON DELETE CASCADE,
    width       integer NOT NULL CHECK (width > 0),
    height      integer NOT NULL CHECK (height > 0),
    storage_key text    NOT NULL UNIQUE,
    mime_type   text    NOT NULL,
    size_bytes  bigint  NOT NULL CHECK (size_bytes >= 0),
    PRIMARY KEY (media_id, width)
);

CREATE TABLE media_alt_texts (
    media_id bigint NOT NULL REFERENCES media (id) ON DELETE CASCADE,
    lang     lang   NOT NULL,
    alt      text   NOT NULL,
    PRIMARY KEY (media_id, lang)
);

-- Posts

CREATE TABLE posts (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    status         post_status NOT NULL DEFAULT 'draft',
    publish_at     timestamptz,
    unpublished_at timestamptz,
    cover_media_id bigint REFERENCES media (id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (status <> 'published' OR publish_at IS NOT NULL)
);

CREATE INDEX posts_status_publish_at_idx ON posts (status, publish_at);
CREATE INDEX posts_cover_media_id_idx ON posts (cover_media_id);

CREATE TABLE post_translations (
    post_id         bigint      NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    lang            lang        NOT NULL,
    slug            text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    title           text        NOT NULL,
    excerpt         text        NOT NULL DEFAULT '',
    body_md         text        NOT NULL DEFAULT '',
    body_html       text        NOT NULL DEFAULT '',
    reading_minutes integer     NOT NULL DEFAULT 0 CHECK (reading_minutes >= 0),
    seo_title       text,
    seo_description text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, lang),
    UNIQUE (lang, slug)
);

CREATE TABLE post_slug_history (
    lang       lang        NOT NULL,
    slug       text        NOT NULL CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    post_id    bigint      NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (lang, slug)
);

CREATE INDEX post_slug_history_post_id_idx ON post_slug_history (post_id);

-- Contexts

CREATE TABLE contexts (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug           text        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    cover_media_id bigint REFERENCES media (id),
    position       integer     NOT NULL,
    visible        boolean     NOT NULL DEFAULT true,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    -- Deferred so a reorder can swap positions inside one transaction.
    CONSTRAINT contexts_position_key UNIQUE (position) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX contexts_cover_media_id_idx ON contexts (cover_media_id);

CREATE TABLE context_translations (
    context_id  bigint NOT NULL REFERENCES contexts (id) ON DELETE CASCADE,
    lang        lang   NOT NULL,
    title       text   NOT NULL,
    description text   NOT NULL DEFAULT '',
    PRIMARY KEY (context_id, lang)
);

-- post_id is the key, so a post belongs to at most one context.
CREATE TABLE context_posts (
    post_id    bigint  PRIMARY KEY REFERENCES posts (id) ON DELETE CASCADE,
    context_id bigint  NOT NULL REFERENCES contexts (id) ON DELETE CASCADE,
    position   integer NOT NULL,
    CONSTRAINT context_posts_context_id_position_key
        UNIQUE (context_id, position) DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE context_slug_history (
    slug       text        PRIMARY KEY CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    context_id bigint      NOT NULL REFERENCES contexts (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX context_slug_history_context_id_idx ON context_slug_history (context_id);

-- Tags

CREATE TABLE tags (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug       text        NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE tag_labels (
    tag_id bigint NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    lang   lang   NOT NULL,
    label  text   NOT NULL,
    PRIMARY KEY (tag_id, lang)
);

CREATE TABLE post_tags (
    post_id bigint NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    tag_id  bigint NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

CREATE INDEX post_tags_tag_id_idx ON post_tags (tag_id);

-- Pages

CREATE TABLE pages (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    key        text        NOT NULL UNIQUE CHECK (key ~ '^[a-z0-9]+(-[a-z0-9]+)*$'),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE page_translations (
    page_id         bigint      NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    lang            lang        NOT NULL,
    title           text        NOT NULL,
    body_md         text        NOT NULL DEFAULT '',
    body_html       text        NOT NULL DEFAULT '',
    seo_title       text,
    seo_description text,
    -- Page-specific extras, such as the CV's email and location.
    fields          jsonb       NOT NULL DEFAULT '{}',
    updated_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (page_id, lang)
);

-- One attached file (the CV PDF) per page and language.
CREATE TABLE page_files (
    page_id  bigint NOT NULL REFERENCES pages (id) ON DELETE CASCADE,
    lang     lang   NOT NULL,
    media_id bigint NOT NULL REFERENCES media (id),
    PRIMARY KEY (page_id, lang)
);

CREATE INDEX page_files_media_id_idx ON page_files (media_id);

-- Users and sessions

CREATE TABLE users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    username      text        NOT NULL,
    password_hash text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_username_key ON users (lower(username));

CREATE TABLE sessions (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      bigint      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash   bytea       NOT NULL UNIQUE,
    user_agent   text        NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);

-- Settings

CREATE TABLE settings (
    key        text        PRIMARY KEY,
    value      jsonb       NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down

DROP TABLE settings;
DROP TABLE sessions;
DROP TABLE users;
DROP TABLE page_files;
DROP TABLE page_translations;
DROP TABLE pages;
DROP TABLE post_tags;
DROP TABLE tag_labels;
DROP TABLE tags;
DROP TABLE context_slug_history;
DROP TABLE context_posts;
DROP TABLE context_translations;
DROP TABLE contexts;
DROP TABLE post_slug_history;
DROP TABLE post_translations;
DROP TABLE posts;
DROP TABLE media_alt_texts;
DROP TABLE media_variants;
DROP TABLE media;
DROP TYPE post_status;
DROP TYPE lang;
