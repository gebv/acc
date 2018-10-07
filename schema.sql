CREATE SCHEMA IF NOT EXISTS acca;

CREATE EXTENSION ltree;

-- money in the numeric(69,18)
-- for example, to store balances for WEI (ETH)
-- change manually if you need less accuracy to

-- currencies
-- currencies any format
CREATE TABLE acca.currencies (
    curr varchar(10) NOT NULL PRIMARY KEY,
    meta jsonb NOT NULL DEFAULT '{}'
);

COMMENT ON COLUMN acca.currencies.curr IS 'Currency ID.';
COMMENT ON COLUMN acca.currencies.meta IS 'Container with meta information.';

-- accounts
-- account and related meta information and current balance
CREATE TABLE acca.accounts (
    acc_id ltree NOT NULL PRIMARY KEY,
    curr varchar(10) REFERENCES acca.currencies(curr),
    balance numeric(69, 18) NOT NULL DEFAULT 0 CHECK(balance >= 0),
    meta jsonb NOT NULL DEFAULT '{}'
);
CREATE INDEX accounts_acc_gist_idx ON acca.accounts USING GIST (acc_id);

COMMENT ON COLUMN acca.accounts.acc_id IS 'Account ID.';
COMMENT ON COLUMN acca.accounts.curr IS 'Currency of account.';
COMMENT ON COLUMN acca.accounts.balance IS 'Current balance.';
COMMENT ON COLUMN acca.accounts.meta IS 'Container with meta information.';


-- transaction status
-- diagram of the transition of the statuses (see TODO: link to WIKI)
CREATE TYPE acca.transaction_status AS enum (
    'unknown',
    'draft',
    'auth',
    'accepted',
    'rejected',

    'failed'
);

-- ALTER TYPE  acca.transaction_status ADD VALUE 'failed';

-- transactions
-- transactions and related meta information and current status
CREATE TABLE acca.transactions (
    tx_id bigserial PRIMARY KEY,
    reason ltree NOT NULL,
    meta jsonb NOT NULL DEFAULT '{}',
    status acca.transaction_status NOT NULL DEFAULT 'unknown' CHECK (status <> 'unknown'),
    errm text,
    created_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone
);
CREATE INDEX transactions_reason_gist_idx ON acca.transactions USING GIST (reason);

COMMENT ON COLUMN acca.transactions.tx_id IS 'Transaction ID.';
COMMENT ON COLUMN acca.transactions.reason IS 'The reason for the transfer.';
COMMENT ON COLUMN acca.transactions.meta IS 'Container with meta information.';
COMMENT ON COLUMN acca.transactions.status IS 'Transaction status.';

-- type of operation
-- - internal - transfer between accounts
-- - recharge - entering funds into the system
-- - withdraw - output funds from the system
CREATE TYPE acca.operation_type AS enum (
    'unknown',
    'internal',
    'recharge',
    'withdraw'
);

-- status of operation
-- diagram of the transition of the statuses (see TODO: link to WIKI)
CREATE TYPE acca.operation_status AS enum (
    'unknown',
    'draft',
    'hold',
    'accepted',
    'rejected'
);

-- operation included in the transaction
CREATE TABLE acca.operations (
    oper_id bigserial PRIMARY KEY,
    tx_id bigint NOT NULL REFERENCES acca.transactions (tx_id),
    src_acc_id ltree NOT NULL REFERENCES acca.accounts(acc_id),
    dst_acc_id ltree NOT NULL REFERENCES acca.accounts(acc_id),
    type acca.operation_type NOT NULL DEFAULT 'unknown' CHECK (type <> 'unknown'),
    amount numeric(69,18) NOT NULL,
    reason ltree NOT NULL DEFAULT '',
    meta jsonb NOT NULL DEFAULT '{}',
    hold boolean NOT NULL DEFAULT false,
    hold_acc_id ltree REFERENCES acca.accounts(acc_id),
    status acca.operation_status NOT NULL DEFAULT 'unknown' CHECK (type <> 'unknown'),
    created_at timestamp without time zone NOT NULL DEFAULT now(),
    updated_at timestamp without time zone
);
CREATE INDEX operations_reason_gist_idx ON acca.operations USING GIST (reason);

COMMENT ON COLUMN acca.operations.src_acc_id IS 'Withdrawal account.';
COMMENT ON COLUMN acca.operations.dst_acc_id IS 'Deposit account.';
COMMENT ON COLUMN acca.operations.type IS 'Type of operation.';
COMMENT ON COLUMN acca.operations.amount IS 'Transaction amount.';
COMMENT ON COLUMN acca.operations.reason IS 'The reason for the operation.';
COMMENT ON COLUMN acca.operations.meta IS 'Container with meta information.';
COMMENT ON COLUMN acca.operations.hold IS 'If true, the translation is two-step.';
COMMENT ON COLUMN acca.operations.hold_acc_id IS 'Suspense account. Only for two-step transaction. May be NULL.';
COMMENT ON COLUMN acca.operations.status IS 'Operation status.';

CREATE TYPE acca.request_type AS enum (
    'unknown',
    'auth',
    'accept',
    'reject'
);

-- request for action
CREATE TABLE acca.requests (
    tx_id bigserial REFERENCES acca.transactions (tx_id),
    type request_type NOT NULL DEFAULT 'unknown' CHECK (type <> 'unknown'),
    created_at timestamp without time zone NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uniq_request_type_for_tx_idx ON requests (tx_id, type);

-- history requests
CREATE TABLE acca.history_requests (
    tx_id bigserial REFERENCES acca.transactions (tx_id),
    type request_type NOT NULL DEFAULT 'unknown' CHECK (type <> 'unknown'),
    created_at timestamp without time zone NOT NULL,
    executed_at timestamp without time zone NOT NULL DEFAULT now()
);

-- TODO: balance_changes

