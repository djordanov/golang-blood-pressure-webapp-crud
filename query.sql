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
;

-- name: CreateBloodPressureObservation :one
INSERT INTO blood_pressure_observation
    (observed_at, systolic, diastolic, pulse, irregular, comment)
VALUES
    ($1, $2, $3, $4, $5, $6)
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
RETURNING *
;

-- name: DeleteBloodPressureObservation :exec
DELETE FROM blood_pressure_observation
WHERE id = $1
;