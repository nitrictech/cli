CREATE TABLE my_migration_table (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL DEFAULT ''
);

-- seed some data
INSERT INTO my_migration_table (name) VALUES ('my-db-foo');
INSERT INTO my_migration_table (name) VALUES ('my-db-bar');