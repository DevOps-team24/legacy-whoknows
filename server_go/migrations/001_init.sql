-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL
);

CREATE TABLE pages (
    title TEXT PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL CHECK (language IN ('en', 'da')) DEFAULT 'en',
    last_updated TIMESTAMPTZ,
    content TEXT NOT NULL
);

-- +goose Down
DROP TABLE pages;
DROP TABLE users;
