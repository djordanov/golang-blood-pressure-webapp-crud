CREATE TABLE blood_pressure_observation (
    id SERIAL PRIMARY KEY,
    observed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    systolic INTEGER NOT NULL,
    diastolic INTEGER NOT NULL,
    pulse INTEGER NOT NULL,
    irregular BOOLEAN NOT NULL DEFAULT false,
    comment TEXT NOT NULL DEFAULT ''
);
