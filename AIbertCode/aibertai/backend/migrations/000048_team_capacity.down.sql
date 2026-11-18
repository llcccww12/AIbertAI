BEGIN;
DROP TRIGGER users_team_capacity_usage ON users;
DROP FUNCTION update_team_capacity_usage();
DROP TABLE team_capacity;
COMMIT;
