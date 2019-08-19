CREATE TABLE acca.invoice_transactions_ext_orders
(
    order_number        VARCHAR PRIMARY KEY,
    payment_system_name VARCHAR                  NOT NULL,
    raw_order_status    VARCHAR                  NOT NULL,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL,
    ext_updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE acca.invoice_transactions_ext_orders IS 'Таблица обновления статусов во внешней системе, например сбербанк.';

COMMENT ON COLUMN acca.invoice_transactions_ext_orders.payment_system_name IS 'Название платежной системы.';
COMMENT ON COLUMN acca.invoice_transactions_ext_orders.order_number IS 'Индентификатор ордера внешней платежной системы.';
COMMENT ON COLUMN acca.invoice_transactions_ext_orders.raw_order_status IS 'Статус ордера внешней платежной системы (исходњое значение).';
COMMENT ON COLUMN acca.invoice_transactions_ext_orders.created_at IS 'Дата создания записи о транзакции во внешнюю систему.';
COMMENT ON COLUMN acca.invoice_transactions_ext_orders.updated_at IS 'Дата обновления данных транзакции во внешней системе.';
COMMENT ON COLUMN acca.invoice_transactions_ext_orders.ext_updated_at IS 'Дата обновления счета в моем деле.';

