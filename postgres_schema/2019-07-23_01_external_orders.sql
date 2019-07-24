CREATE TYPE payment_system_name AS enum (
    'internal',
    'sberbank'
    );

CREATE TABLE invoice_transactions_ext_orders
(
    payment_system_name payment_system_name      NOT NULL,
    order_number        VARCHAR PRIMARY KEY,
    raw_order_status    VARCHAR                  NOT NULL,
    order_status        VARCHAR                  NOT NULL,
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL,
    ext_updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE invoice_transactions_ext_orders IS 'Таблица обновления статусов во внешней системе, например сбербанк.';

COMMENT ON COLUMN invoice_transactions_ext_orders.payment_system_name IS 'Название платежной системы.';
COMMENT ON COLUMN invoice_transactions_ext_orders.order_number IS 'Индентификатор ордера внешней платежной системы.';
COMMENT ON COLUMN invoice_transactions_ext_orders.raw_order_status IS 'Статус ордера внешней платежной системы (исходњое значение).';
COMMENT ON COLUMN invoice_transactions_ext_orders.order_status IS 'Значение статуса мапнутое с внутренним значением статуса.';
COMMENT ON COLUMN invoice_transactions_ext_orders.created_at IS 'Дата создания записи о транзакции во внешнюю систему.';
COMMENT ON COLUMN invoice_transactions_ext_orders.updated_at IS 'Дата обновления данных транзакции во внешней системе.';
COMMENT ON COLUMN invoice_transactions_ext_orders.ext_updated_at IS 'Дата обновления счета в моем деле.';

