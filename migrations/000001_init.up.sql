CREATE TABLE IF NOT EXISTS users
(
    user_id    BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    username   VARCHAR(255) NOT NULL,
    password   TEXT         NOT NULL,
    created_at TIMESTAMP    NOT NULL,
    updated_at TIMESTAMP    NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idxu_users_username ON users (username);

CREATE TABLE IF NOT EXISTS entity
(
    entity_id  BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id    INT       NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    data       BYTEA     NOT NULL,
    metadata   JSONB     NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_entity_user_id ON entity (user_id);
CREATE INDEX IF NOT EXISTS idx_entity_metadata ON entity (metadata);

CREATE TABLE IF NOT EXISTS file
(
    file_id    BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id    INT          NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMP    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_file_user_id ON file (user_id);

CREATE TABLE IF NOT EXISTS access_token
(
    access_token VARCHAR(100) NOT NULL,
    user_id      INT          NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    iat          TIMESTAMP    NOT NULL,
    exp          TIMESTAMP    NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idxu_access_token_access_token ON access_token (access_token);
CREATE INDEX IF NOT EXISTS idx_access_token_user_id ON access_token (user_id);
CREATE INDEX IF NOT EXISTS idx_access_token_exp ON access_token (exp);
