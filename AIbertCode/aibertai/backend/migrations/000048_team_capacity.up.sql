BEGIN;

LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE;

CREATE TABLE team_capacity (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    max_members integer CHECK (max_members > 0),
    active_members integer NOT NULL CHECK (active_members >= 0),
    CHECK (max_members IS NULL OR active_members <= max_members)
);

INSERT INTO team_capacity (active_members)
SELECT count(*) FROM users WHERE status = 'active' AND deleted_at IS NULL;

CREATE FUNCTION update_team_capacity_usage() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    delta integer := 0;
BEGIN
    IF TG_OP <> 'INSERT' AND OLD.status = 'active' AND OLD.deleted_at IS NULL THEN
        delta := delta - 1;
    END IF;
    IF TG_OP <> 'DELETE' AND NEW.status = 'active' AND NEW.deleted_at IS NULL THEN
        delta := delta + 1;
    END IF;
    IF delta <> 0 THEN
        UPDATE team_capacity
        SET active_members = active_members + delta
        WHERE singleton AND (max_members IS NULL OR active_members + delta <= max_members);
        IF NOT FOUND THEN
            RAISE EXCEPTION 'Team member limit reached' USING ERRCODE = '23514', CONSTRAINT = 'team_capacity_limit';
        END IF;
    END IF;
    RETURN NULL;
END;
$$;

CREATE TRIGGER users_team_capacity_usage
AFTER INSERT OR UPDATE OF status, deleted_at OR DELETE ON users
FOR EACH ROW EXECUTE FUNCTION update_team_capacity_usage();

COMMIT;
