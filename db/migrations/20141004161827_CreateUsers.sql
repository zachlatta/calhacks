
-- +goose Up
CREATE TABLE users (
  id serial primary key not null,
  created timestamp not null,
  updated timestamp not null,
  username text not null,
  profile_picture text not null,
  github_id integer not null unique,
  github_url text not null,
  access_token text not null
);


-- +goose Down
DROP TABLE users;

