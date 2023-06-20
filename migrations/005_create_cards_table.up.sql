CREATE TABLE cards (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    stripe_pay_method_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
