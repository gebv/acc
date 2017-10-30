CREATE SCHEMA IF NOT EXISTS shop;

CREATE TABLE shop.orders (
    order_id text NOT NULL PRIMARY KEY,
    destination_id bigint NOT NULL REFERENCES finances.accounts(account_id),
    order_type text NOT NULL,
    total bigint NOT NULL CHECK (total >= 0),
    closed boolean NOT NULL default false,
    created_at timestamp with time zone NOT NULL
);