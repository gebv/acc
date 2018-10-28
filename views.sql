CREATE OR REPLACE VIEW acca.recent_activity AS
    SELECT
        bc.ch_id as id,
        bc.oper_id as oper_id,
        bc.acc_id as acc_id,
        bc.amount as amount,
        bc.balance as balance,

        -- operations
        o.tx_id as tx_id,
        o.src_acc_id as src_acc_id,
        o.dst_acc_id as dst_acc_id,
        -- o.type as type,
        o.reason as reason,
        -- o.meta as meta,
        -- o.hold as hold,
        -- o.hold_acc_id as hold_acc_id,
        -- o.status as status,

        -- transactions
        t.reason AS tx_reason,

        -- accounts
        a.key AS acc_key,

        -- currency
        c.curr_id AS acc_curr_id,
        c.key AS acc_curr_key

    FROM acca.balance_changes bc
    LEFT JOIN acca.operations o USING(oper_id)
    LEFT JOIN acca.transactions t USING(tx_id)
    LEFT JOIN acca.accounts a USING(acc_id)
    LEFT JOIN acca.currencies c USING(curr_id)
    ORDER BY id DESC;
