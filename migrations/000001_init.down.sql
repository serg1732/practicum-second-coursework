DROP INDEX IF EXISTS idx_access_token_exp;
DROP INDEX IF EXISTS idx_access_token_user_id;
DROP INDEX IF EXISTS idxu_access_token_access_token;
DROP TABLE IF EXISTS access_token;

DROP INDEX IF EXISTS idx_file_user_id;
DROP TABLE IF EXISTS file;

DROP INDEX IF EXISTS idx_entity_metadata;
DROP INDEX IF EXISTS idx_entity_user_id;
DROP TABLE IF EXISTS entity;

DROP INDEX IF EXISTS idxu_users_username;
DROP TABLE IF EXISTS users;
