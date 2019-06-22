BEGIN;

CREATE SCHEMA IF NOT EXISTS acca;

CREATE EXTENSION IF NOT EXISTS ltree;

CREATE TYPE acca.invoice_status AS enum (
    'unknown',
    'draft',
    'auth',
    'wait',
    'accepted',
    'rejected'
);

CREATE TABLE acca.invoices (
    invoice_id bigserial PRIMARY KEY,
    key ltree NOT NULL,
    status acca.invoice_status NOT NULL CHECK (status <> 'unknown'),
    strategy varchar NOT NULL,
    total_amount numeric(23, 00) NOT NULL DEFAULT 0,
    meta jsonb,
    payload bytea,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at timestamp with time zone NOT NULL
);

CREATE TABLE acca.currencies (
    curr_id bigserial PRIMARY KEY,
    key ltree NOT NULL,
    meta jsonb
);
CREATE INDEX currencies_key_gist_idx ON acca.currencies USING GIST (key);
CREATE UNIQUE INDEX currencies_key_uniq_idx ON acca.currencies (key);

CREATE TABLE acca.accounts (
    acc_id bigserial PRIMARY KEY,
    curr_id bigint NOT NULL  REFERENCES acca.currencies(curr_id),
    key ltree NOT NULL,
    balance numeric(23, 00) NOT NULL DEFAULT 0,
    balance_accepted numeric(23, 00) NOT NULL DEFAULT 0,
    meta jsonb,
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at timestamp with time zone NOT NULL
);
CREATE INDEX accounts_key_gist_idx ON acca.accounts USING GIST (key);
CREATE UNIQUE INDEX accounts_curr_key_uniq_idx ON acca.accounts (curr_id, key);

CREATE TYPE acca.tx_status AS enum (
    'unknown',
    'draft',
    'auth',
    'accepted',
    'rejected',

    'failed'
);

CREATE TABLE acca.transactions (
    tx_id bigserial PRIMARY KEY,
    invoice_id bigint NOT NULL REFERENCES acca.invoices(invoice_id),
    key ltree,
    provider varchar NOT NULL,
    provider_oper_id varchar,
    provider_oper_status varchar,
    meta jsonb,
    status acca.tx_status NOT NULL CHECK (status <> 'unknown'),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at timestamp with time zone NOT NULL
);
CREATE INDEX transactions_key_gist_idx ON acca.transactions USING GIST (key);
CREATE UNIQUE INDEX transaction_invoice_key_uniq_idx ON acca.transactions (invoice_id, key);


CREATE TYPE acca.operation_strategy AS enum (
    'unknown',
    'simple_transfer', -- src to dst
    'recharge', -- both increase
    'withdraw' -- both decrease
);

CREATE TYPE acca.operation_status AS enum (
    'unknown',
    'draft',
    'hold',
    'accepted',
    'rejected'
);

CREATE TABLE acca.operations (
    oper_id bigserial PRIMARY KEY,
    invoice_id bigint NOT NULL REFERENCES acca.invoices (invoice_id),
    tx_id bigint NOT NULL REFERENCES acca.transactions (tx_id),
    src_acc_id bigint NOT NULL REFERENCES acca.accounts(acc_id),
    hold boolean NOT NULL DEFAULT false,
    hold_acc_id bigint REFERENCES acca.accounts(acc_id),
    dst_acc_id bigint NOT NULL REFERENCES acca.accounts(acc_id),
    strategy acca.operation_strategy NOT NULL CHECK (strategy <> 'unknown'),
    amount numeric(23, 00) NOT NULL,
    key ltree,
    meta jsonb,
    status acca.operation_status NOT NULL CHECK (status <> 'unknown'),
    updated_at timestamp with time zone NOT NULL DEFAULT now(),
    created_at timestamp with time zone NOT NULL
);

CREATE INDEX operations_key_gist_idx ON acca.operations USING GIST (key);

COMMIT;
