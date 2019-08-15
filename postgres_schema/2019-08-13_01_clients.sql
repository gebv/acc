BEGIN;

CREATE TABLE acca.clients
(
    client_id    BIGSERIAL PRIMARY KEY,
    access_token VARCHAR                  NOT NULL,
    created_at   timestamp with time zone NOT NULL
);

DROP INDEX currencies_key_gist_idx;
ALTER TABLE currencies
    ALTER COLUMN key TYPE VARCHAR USING key::VARCHAR,
    ADD CONSTRAINT len_key CHECK ( length(key) > 2);
ALTER TABLE currencies
    ADD client_id BIGINT REFERENCES acca.clients (client_id);
DROP INDEX currencies_key_uniq_idx;
CREATE UNIQUE INDEX currencies_key_uniq_idx
    ON currencies (client_id, key);

ALTER TABLE acca.accounts
    ADD COLUMN client_id BIGINT REFERENCES acca.clients (client_id);

ALTER TABLE acca.balance_changes
    ADD COLUMN client_id BIGINT REFERENCES acca.clients (client_id);


-- trigger for create new record to balance changes table after new record in operations table
CREATE OR REPLACE FUNCTION add_balance_changes() RETURNS trigger AS
$add_balance_changes$
DECLARE
    _amount      numeric(69, 00);
    _invoice     jsonb;
    _transaction jsonb;
    _opers       jsonb;
BEGIN
    IF NEW.balance = OLD.balance AND NEW.balance_accepted = OLD.balance_accepted THEN
        RETURN NEW;
    END IF;

    IF NEW.last_tx_id IS NULL THEN
        RAISE EXCEPTION 'last_tx_id cannot be null if changes balance';
    END IF;

    IF NEW.balance != OLD.balance THEN
        _amount = NEW.balance - OLD.balance;
    ELSEIF NEW.balance_accepted != OLD.balance_accepted THEN
        _amount = NEW.balance_accepted - OLD.balance_accepted;
    END IF;

    SELECT json_build_object(
                   'invoice_id', i.invoice_id,
                   'key', i.key,
                   'meta', i.meta,
                   'strategy', i.strategy,
                   'status', i.status)::jsonb,
           json_build_object(
                   'tx_id', t.tx_id,
                   'key', t.key,
                   'meta', t.meta,
                   'strategy', t.strategy,
                   'status', t.status,
                   'provider', t.provider,
                   'provider_oper_id', t.provider_oper_id,
                   'provider_oper_status', t.provider_oper_status,
                   'provider_oper_url', t.provider_oper_url)::jsonb,
           array_to_json(array_agg(json_build_object(
                   'oper_id', o.oper_id,
                   'src_acc_id', o.src_acc_id,
                   'dst_acc_id', o.dst_acc_id,
                   'amount', o.amount,
                   'strategy', o.strategy,
                   'key', o.key,
                   'meta', o.meta,
                   'hold', o.hold,
                   'hold_acc_id', o.hold_acc_id,
                   'status', o.status)))::jsonb
    INTO _invoice, _transaction,_opers
    FROM acca.transactions t
             INNER JOIN acca.invoices i USING (invoice_id)
             INNER JOIN acca.operations o USING (tx_id)
    WHERE t.tx_id = NEW.last_tx_id
    GROUP BY 1, 2;

    INSERT INTO acca.balance_changes(client_id, tx_id, acc_id, curr_id, amount, balance, balance_accepted, invoice,
                                     transaction,
                                     operations)
    VALUES (NEW.client_id, NEW.last_tx_id, NEW.acc_id, NEW.curr_id, _amount, NEW.balance, NEW.balance_accepted,
            _invoice, _transaction,
            _opers);

    RETURN NEW;
END;
$add_balance_changes$ LANGUAGE plpgsql;

DROP VIEW acca.view_balance_changes;

CREATE OR REPLACE VIEW acca.view_balance_changes AS
SELECT bc.*,
       json_build_object(
               'acc_id', bc.acc_id,
               'key', a.key,
               'balance', a.balance,
               'balance_accepted', a.balance_accepted)::jsonb   AS actual_account,
       json_build_object(
               'tx_id', t.tx_id,
               'key', t.key,
               'meta', t.meta,
               'strategy', t.strategy,
               'status', t.status,
               'provider', t.provider,
               'provider_oper_id', t.provider_oper_id,
               'provider_oper_status', t.provider_oper_status,
               'provider_oper_url', t.provider_oper_url)::jsonb AS actual_transaction
FROM acca.balance_changes AS bc
         INNER JOIN acca.accounts a USING (acc_id)
         INNER JOIN acca.transactions t USING (tx_id)
ORDER BY tx_id DESC, ch_id DESC;

ALTER TABLE acca.invoices
    ADD COLUMN client_id BIGINT REFERENCES acca.clients (client_id);

ALTER TABLE acca.transactions
    ADD COLUMN client_id BIGINT REFERENCES acca.clients (client_id);

DROP VIEW acca.v_invoices;
DROP VIEW acca.v_transactions;

CREATE OR REPLACE VIEW acca.v_transactions as
SELECT t.*,
       JSON_AGG(
               JSON_BUILD_OBJECT(
                       'oper_id', o.oper_id,
                       'tx_id', o.tx_id,
                       'invoice_id', o.invoice_id,
                       'src_acc_id', o.src_acc_id,
                       'dst_acc_id', o.dst_acc_id,
                       'strategy', o.strategy,
                       'amount', o.amount,
                       'key', o.key,
                       'meta', o.meta,
                       'hold', o.hold,
                       'hold_acc_id', o.hold_acc_id,
                       'status', o.status,
                       'created_at', o.created_at,
                       'updated_at', o.updated_at
                   )
           ) AS operations
FROM transactions AS t
         INNER JOIN operations AS o USING (tx_id)
GROUP BY 1;

CREATE OR REPLACE VIEW acca.v_invoices as
SELECT i.*,
       JSON_AGG(
               JSON_BUILD_OBJECT(
                       'tx_id', t.tx_id,
                       'invoice_id', t.invoice_id,
                       'key', t.key,
                       'strategy', t.strategy,
                       'amount', t.amount,
                       'provider', t.provider,
                       'provider_oper_id', t.provider_oper_id,
                       'provider_oper_status', t.provider_oper_status,
                       'provider_oper_url', t.provider_oper_url,
                       'meta', t.meta,
                       'status', t.status,
                       'next_status', t.next_status,
                       'updated_at', t.updated_at,
                       'created_at', t.created_at,
                       'operations', t.operations
                   )
           ) AS transactions
FROM invoices AS i
         INNER JOIN v_transactions AS t USING (invoice_id)
GROUP BY 1;


COMMIT;
