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
    (:observed_at, :systolic, :diastolic, :pulse, :irregular, :comment)
RETURNING *
;

-- name: UpdateBloodPressureObservation :one
UPDATE blood_pressure_observation
SET
    observed_at = :observed_at,
    systolic = :systolic,
    diastolic = :diastolic,
    pulse = :pulse,
    irregular = :irregular,
    comment = :comment
WHERE id = :id
RETURNING *
;

-- name: DeleteBloodPressureObservation :exec
DELETE FROM blood_pressure_observation
WHERE id = ?
;