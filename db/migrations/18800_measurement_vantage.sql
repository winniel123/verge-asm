-- +goose Up
-- A Vantage is a network position observations are made from (CONTEXT.md
-- `Vantage`). The recursive resolver it resolves through is part of that
-- position and therefore part of the term's identity (ADR-0070), so it is held
-- on the row and never as a leaf parameter. Class is re-verified every batch;
-- it ships `unverified` until a prober observes the instance's presented
-- address (v1 spec §4.2), which is the honest default for a name-scope-only
-- install with no prober.
CREATE TABLE vantage (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    class      TEXT NOT NULL DEFAULT 'unverified' CHECK (class IN ('internet', 'internal', 'unverified')),
    resolver   TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One local Vantage ships so a name-scope install resolves at the dns cadence
-- from the first run. Its resolver is operator-editable; the shipped value is a
-- placeholder the operator replaces with their own recursive resolver.
INSERT INTO vantage (name, class, resolver) VALUES ('local', 'unverified', '127.0.0.1:53');

-- +goose Down
DROP TABLE vantage;
