
-- +goose Up
CREATE TABLE challenges (
  id serial not null primary key,
  created timestamp not null,
  updated timestamp not null,
  title text not null,
  description text not null,
  seconds integer not null
);

CREATE TABLE challenge_test_cases (
  id serial not null primary key,
  created timestamp not null,
  updated timestamp not null,
  challenge_id integer references challenges(id) not null
);


-- +goose Down
DROP TABLE challenge_test_cases;
DROP TABLE challenges;

