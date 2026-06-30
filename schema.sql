CREATE TABLE person (
    id SERIAL PRIMARY KEY,
    email VARCHAR UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE session (
    id UUID PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES person(id),
    expires_at TIMESTAMP NOT NULL
);

CREATE TABLE blood_pressure_observation (
    id SERIAL PRIMARY KEY,
    person_id INTEGER NOT NULL REFERENCES person(id),
    observed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    systolic INTEGER NOT NULL,
    diastolic INTEGER NOT NULL,
    pulse INTEGER NOT NULL,
    irregular BOOLEAN NOT NULL DEFAULT false,
    comment TEXT NOT NULL DEFAULT ''
);
