CREATE TABLE trainings (
    id SERIAL PRIMARY KEY,
    sequence INTEGER NOT NULL,
    topic VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(255) NOT NULL,
    is_free BOOLEAN NOT NULL,
    project_url VARCHAR(255),
    course_id INTEGER REFERENCES courses(id) NOT NULL
);
