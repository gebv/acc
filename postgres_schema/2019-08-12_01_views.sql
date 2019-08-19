BEGIN;

ALTER TABLE acca.invoices
    DROP COLUMN amount;

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
FROM acca.transactions AS t
         INNER JOIN acca.operations AS o USING (tx_id)
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
FROM acca.invoices AS i
         INNER JOIN acca.v_transactions AS t USING (invoice_id)
GROUP BY 1;

COMMIT;