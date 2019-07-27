BEGIN;

ALTER TABLE acca.accounts
    ADD COLUMN last_tx_id bigint REFERENCES acca.transactions (tx_id);
COMMENT ON COLUMN acca.accounts.last_tx_id IS 'Related last transaction changing balance (last_tx_id must not null if changes balance).';

-- balance changes table
CREATE TABLE acca.balance_changes
(
    ch_id            bigserial PRIMARY KEY,
    tx_id            bigint          NOT NULL REFERENCES acca.transactions (tx_id),
    acc_id           bigint          NOT NULL REFERENCES acca.accounts (acc_id),
    curr_id          bigint          NOT NULL REFERENCES acca.currencies (curr_id),
    amount           numeric(69, 00) NOT NULL,
    balance          numeric(69, 00) NOT NULL,
    balance_accepted numeric(69, 00) NOT NULL,
    invoice          jsonb,
    transaction      jsonb,
    operations       jsonb
);

COMMENT ON COLUMN acca.balance_changes.ch_id IS 'Change ID.';
COMMENT ON COLUMN acca.balance_changes.tx_id IS 'Related transaction.';
COMMENT ON COLUMN acca.balance_changes.acc_id IS 'Related account.';
COMMENT ON COLUMN acca.balance_changes.amount IS 'Transaction amount.';
COMMENT ON COLUMN acca.balance_changes.balance IS 'Balance after transaction.';
COMMENT ON COLUMN acca.balance_changes.balance_accepted IS 'Accepted balance.';


-- trigger for create new record to balance changes table after new record in operations table
CREATE FUNCTION add_balance_changes() RETURNS trigger AS
$add_balance_changes$
DECLARE
    _amount      numeric(69, 00);
    _invoice     jsonb;
    _transaction jsonb;
    _opers       jsonb;
BEGIN
    IF NEW.balance = OLD.balance THEN
        RETURN NEW;
    END IF;

    IF NEW.last_tx_id IS NULL THEN
        RAISE EXCEPTION 'last_tx_id cannot be null if changes balance';
    END IF;

    _amount := NEW.balance - OLD.balance;

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

    INSERT INTO acca.balance_changes(tx_id, acc_id, curr_id, amount, balance, balance_accepted, invoice, transaction,
                                     operations)
    VALUES (NEW.last_tx_id, NEW.acc_id, NEW.curr_id, _amount, NEW.balance, NEW.balance_accepted, _invoice, _transaction,
            _opers);

    RETURN NEW;
END;
$add_balance_changes$ LANGUAGE plpgsql;

CREATE TRIGGER b_add_balance_changes_trigger
    AFTER UPDATE
    ON acca.accounts
    FOR EACH ROW
EXECUTE PROCEDURE add_balance_changes();

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


COMMIT;