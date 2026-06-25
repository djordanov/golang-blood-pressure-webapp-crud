-- name: GetSessionById :one
SELECT
    id,
    user,
    expires_at
FROM session
WHERE id = $1
;

-- name: CreateUser :one
INSERT INTO user
    (email, created_at)
VALUES ($1, $2)
RETURNING *
;

-- name: CreateSession :one
INSERT INTO session
    (id, user, expires_at)
VALUES ($1, $2, $3)
RETURNING *
;

-- name: GetBloodPressureObservations :many
SELECT
    id,
    observed_at,
    systolic,
    diastolic,
    pulse,
    irregular,
    comment
FROM blood_pressure_observation
WHERE user = $1
;

-- name: CreateBloodPressureObservation :one
INSERT INTO blood_pressure_observation
    (observed_at, systolic, diastolic, pulse, irregular, comment, user)
VALUES
    ($1, $2, $3, $4, $5, $6, $7)
RETURNING *
;

-- name: UpdateBloodPressureObservation :one
UPDATE blood_pressure_observation
SET
    observed_at = $1,
    systolic = $2,
    diastolic = $3,
    pulse = $4,
    irregular = $5,
    comment = $6
WHERE id = $7
    AND user = $8
RETURNING *
;

-- name: DeleteBloodPressureObservation :exec
DELETE FROM blood_pressure_observation
WHERE id = $1
    AND user = $2
;
