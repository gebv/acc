CREATE SCHEMA IF NOT EXISTS finances;

CREATE TYPE finances.account_type AS ENUM('system', 'partner', 'customer');
CREATE TABLE finances.accounts (
    account_id bigserial PRIMARY KEY,
    customer_id text NOT NULL,
    _type finances.account_type NOT NULL DEFAULT 'customer',
    balance bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE TABLE finances.invoices (
    invoice_id bigserial PRIMARY KEY,
    order_id text NOT NULL,
    destination_id bigint NOT NULL REFERENCES finances.accounts(account_id),
    source_id bigint REFERENCES finances.accounts(account_id),
    paid boolean NOT NULL DEFAULT false,
    amount bigint NOT NULL DEFAULT 0 CHECK (amount > 0),
    created_at timestamp with time zone NOT NULL
);

CREATE TYPE finances.tx_type AS ENUM('authorization', 'accepted', 'rejected');
CREATE TABLE finances.transactions (
    transaction_id bigserial PRIMARY KEY,
    invoice_id bigint NOT NULL REFERENCES finances.invoices(invoice_id),
    amount bigint NOT NULL DEFAULT 0 CHECK (amount > 0),
    source bigint NOT NULL REFERENCES finances.accounts(account_id),
    destination bigint NOT NULL CHECK (destination <> source) REFERENCES finances.accounts(account_id), 
    status finances.tx_type NOT NULL DEFAULT 'authorization',
    created_at timestamp with time zone NOT NULL,
    closed_at timestamp with time zone NOT NULL
);

CREATE TYPE finances.bc_type AS ENUM('hold', 'refund', 'complete');
CREATE TABLE finances.balance_changes (
    change_id bigserial PRIMARY KEY,
    account_id bigint NOT NULL REFERENCES finances.accounts(account_id),
    transaction_id bigint NOT NULL REFERENCES finances.transactions(transaction_id),
    _type finances.bc_type NOT NULL DEFAULT 'hold',
    amount bigint NOT NULL CHECK (amount <> 0),
    balance bigint NOT NULL CHECK(balance >= 0),
    created_at timestamp with time zone NOT NULL
);

