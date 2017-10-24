CREATE TYPE account_type AS ENUM('system', 'partner', 'customer');
CREATE TABLE accounts (
    account_id bigserial PRIMARY KEY,
    customer_id text NOT NULL,
    _type account_type NOT NULL DEFAULT 'customer',
    balance bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE TABLE invoices (
    invoice_id bigserial PRIMARY KEY,
    order_id text NOT NULL,
    destination_id bigint NOT NULL REFERENCES accounts(account_id),
    source_id bigint REFERENCES accounts(account_id),
    paid boolean NOT NULL DEFAULT false,
    amount bigint NOT NULL DEFAULT 0 CHECK (amount > 0),
    created_at timestamp with time zone NOT NULL
);

CREATE TYPE tx_type AS ENUM('authorization', 'accepted', 'rejected');
CREATE TABLE transactions (
    transaction_id bigserial PRIMARY KEY,
    invoice_id bigint NOT NULL REFERENCES invoices(invoice_id),
    amount bigint NOT NULL DEFAULT 0 CHECK (amount > 0),
    source bigint NOT NULL REFERENCES accounts(account_id),
    destination bigint NOT NULL CHECK (destination <> source) REFERENCES accounts(account_id), 
    status tx_type NOT NULL DEFAULT 'authorization',
    created_at timestamp with time zone NOT NULL,
    closed_at timestamp with time zone NOT NULL
);

CREATE TYPE bc_type AS ENUM('hold', 'refund', 'complete');
CREATE TABLE balance_changes (
    change_id bigserial PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES accounts(account_id),
    transaction_id bigint NOT NULL REFERENCES transactions(transaction_id),
    _type bc_type NOT NULL DEFAULT 'hold',
    amount bigint NOT NULL CHECK (amount <> 0),
    balance bigint NOT NULL CHECK(balance >= 0),
    created_at timestamp with time zone NOT NULL
);
