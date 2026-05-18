-- +goose up
ALTER TABLE users
ADD is_chirpy_red BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose down
ALTER TABLE users
DROP is_chirpy_red;