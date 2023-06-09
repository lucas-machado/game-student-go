CREATE TABLE payments (
    id SERIAL PRIMARY KEY,
    stripe_payment_intent_id VARCHAR(255) NOT NULL,
    stripe_pay_method_id VARCHAR(255) NOT NULL,
    user_id INT NOT NULL,
    amount INT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    description TEXT,
    status VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
