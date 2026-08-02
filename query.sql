-- name: GetSessionById :one
SELECT
    id,
    person_id,
    expires_at
FROM session
WHERE id = $1
;

-- name: CreateSession :one
INSERT INTO session
    (id, person_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *
;

-- name: GetPersonByEmail :one
SELECT
    id,
    email,
    created_at
FROM person
WHERE email = $1
;

-- name: CreatePerson :one
INSERT INTO person
    (email, created_at)
VALUES ($1, $2)
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
WHERE person_id = $1
ORDER BY observed_at DESC
LIMIT $2
OFFSET $3
;

-- name: GetBloodPressureObservation :one
SELECT
    id,
    observed_at,
    systolic,
    diastolic,
    pulse,
    irregular,
    comment
FROM blood_pressure_observation
WHERE person_id = $1
    AND id = $2
;

-- name: CreateBloodPressureObservation :one
INSERT INTO blood_pressure_observation
    (observed_at, systolic, diastolic, pulse, irregular, comment, person_id)
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
    AND person_id = $8
RETURNING *
;

-- name: DeleteBloodPressureObservation :exec
DELETE FROM blood_pressure_observation
WHERE id = $1
    AND person_id = $2
;
