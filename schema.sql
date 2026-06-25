CREATE TABLE blood_pressure_observation (
    id SERIAL PRIMARY KEY,
    user REFERENCES user(id) NOT NULL,
    observed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    systolic INTEGER NOT NULL,
    diastolic INTEGER NOT NULL,
    pulse INTEGER NOT NULL,
    irregular BOOLEAN NOT NULL DEFAULT false,
    comment TEXT NOT NULL DEFAULT ''
);

CREATE TABLE user (
    id SERIAL PRIMARY KEY,
    email VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE session (
    id UUID NOT NULL,
    user REFERENCES user(id) NOT NULL,
    expires_at TIMESTAMP NOT NULL
);
