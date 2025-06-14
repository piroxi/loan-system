DROP TABLE IF EXISTS loan_disbursements CASCADE;
DROP TABLE IF EXISTS loan_approvals CASCADE;
DROP TABLE IF EXISTS investments CASCADE;
DROP TABLE IF EXISTS loans CASCADE;
DROP TABLE IF EXISTS users CASCADE;


CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    role INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

INSERT INTO users (id, username, role, created_at, updated_at) VALUES
('1', 'borrower1', 1, NOW(), NOW()),
('2', 'validator', 2, NOW(), NOW()),
('3', 'investor1', 3, NOW(), NOW()),
('4', 'investor2', 3, NOW(), NOW()),
('5', 'disburser', 4, NOW(), NOW());

CREATE TABLE loans (
    id SERIAL PRIMARY KEY,
    borrower_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    principal NUMERIC NOT NULL,
    rate NUMERIC NOT NULL,
    roi NUMERIC NOT NULL,
    status TEXT NOT NULL,
    agreement_link TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE loan_approvals (
    id SERIAL PRIMARY KEY,
    loan_id INT NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    validator_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reject_reason TEXT,
    photo_url TEXT NOT NULL,
    approved_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE loan_disbursements (
    id SERIAL PRIMARY KEY,
    loan_id INT NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    signed_agreement_url TEXT NOT NULL,
    disburser_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    disbursed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE investments (
    id SERIAL PRIMARY KEY,
    loan_id INT NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    investor_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);
