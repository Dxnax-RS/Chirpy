-- +goose up
ALTER TABLE users
ADD hashed_password TEXT NOT NULL;

-- +goose down
ALTER TABLE users
DROP hashed_password;