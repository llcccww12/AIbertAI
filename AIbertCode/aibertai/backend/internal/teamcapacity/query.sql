-- name: GetCapacity :one
SELECT max_members, active_members FROM team_capacity WHERE singleton;

-- name: SetCapacity :one
UPDATE team_capacity
SET max_members = sqlc.narg('max_members')::integer
WHERE singleton AND (sqlc.narg('max_members')::integer IS NULL OR active_members <= sqlc.narg('max_members')::integer)
RETURNING max_members, active_members;

-- name: TouchActiveLogin :execrows
UPDATE users
SET last_login_at = now(), updated_at = now()
WHERE id = $1 AND status = 'active' AND deleted_at IS NULL;

-- name: LockActiveAdmin :exec
SELECT pg_advisory_xact_lock(hashtext('monkeyai:active-admin'));

-- name: IsLastActiveAdmin :one
SELECT (EXISTS (SELECT 1 FROM users WHERE id = sqlc.arg('id')::uuid AND role = 'admin' AND status = 'active' AND deleted_at IS NULL)
   AND NOT EXISTS (SELECT 1 FROM users WHERE id <> sqlc.arg('id')::uuid AND role = 'admin' AND status = 'active' AND deleted_at IS NULL))::boolean;
