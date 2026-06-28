-- name: ResetUsers :exec
DELETE FROM users;

-- name: ResetChirps :exec
DELETE FROM chirps;