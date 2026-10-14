-- name: CountUsers :one
-- CountUsers tells whether the admin still has to be seeded on first start.
SELECT count(*) FROM users;
