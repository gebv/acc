CREATE SCHEMA IF NOT EXISTS activity;

CREATE TABLE activity.http_requests
(
    created_at timestamp without time zone NOT NULL,
    request_id varchar                     NOT NULL,
    method     varchar                     NOT NULL,
    user_id    bigint,
    device_id  varchar,
    body       text,
    user_ip    inet,
    proxy_ip   inet,
    user_agent varchar(255)                NOT NULL,
    session_id VARCHAR
) PARTITION BY RANGE (created_at);

COMMENT ON TABLE activity.http_requests IS 'Таблица с историей пользовательских http запросов (только gRPC).';

-- (на момент написания) значение задает nginx
-- proxy_set_header X-Request-Id $request_id;
-- http://nginx.org/en/docs/http/ngx_http_core_module.html#var_request_id
COMMENT ON COLUMN activity.http_requests.request_id IS 'Идентификатор запроса.';
COMMENT ON COLUMN activity.http_requests.method IS 'gRPC метод с суффиксом обозначающий реквест и респонс.';
COMMENT ON COLUMN activity.http_requests.user_id IS 'Идентификатор пользователя (если есть).';
COMMENT ON COLUMN activity.http_requests.device_id IS 'Идентификатор устройства.';
COMMENT ON COLUMN activity.http_requests.body IS 'Тело запроса или ответа.';
COMMENT ON COLUMN activity.http_requests.user_ip IS 'IP пользователя.';
COMMENT ON COLUMN activity.http_requests.proxy_ip IS 'Первое IP прокси, если есть.';
COMMENT ON COLUMN activity.http_requests.user_agent IS 'User-Agent пользователя.';
COMMENT ON COLUMN activity.http_requests.session_id IS 'Идентификатор сессии.';

CREATE TABLE activity.http_requests_y2019m07 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2019-07-01') TO ('2019-08-01');
CREATE INDEX ON activity.http_requests_y2019m07 (created_at);

CREATE TABLE activity.http_requests_y2019m08 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2019-08-01') TO ('2019-09-01');
CREATE INDEX ON activity.http_requests_y2019m08 (created_at);

CREATE TABLE activity.http_requests_y2019m09 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2019-09-01') TO ('2019-10-01');
CREATE INDEX ON activity.http_requests_y2019m09 (created_at);

CREATE TABLE activity.http_requests_y2019m10 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2019-10-01') TO ('2019-11-01');
CREATE INDEX ON activity.http_requests_y2019m10 (created_at);

CREATE TABLE activity.http_requests_y2019m11 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2019-11-01') TO ('2019-12-01');
CREATE INDEX ON activity.http_requests_y2019m11 (created_at);

CREATE TABLE activity.http_requests_y2019m12 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2019-12-01') TO ('2020-01-01');
CREATE INDEX ON activity.http_requests_y2019m12 (created_at);

CREATE TABLE activity.http_requests_y2020m01 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-01-01') TO ('2020-02-01');
CREATE INDEX ON activity.http_requests_y2020m01 (created_at);

CREATE TABLE activity.http_requests_y2020m02 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-02-01') TO ('2020-03-01');
CREATE INDEX ON activity.http_requests_y2020m02 (created_at);

CREATE TABLE activity.http_requests_y2020m03 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-03-01') TO ('2020-04-01');
CREATE INDEX ON activity.http_requests_y2020m03 (created_at);

CREATE TABLE activity.http_requests_y2020m04 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-04-01') TO ('2020-05-01');
CREATE INDEX ON activity.http_requests_y2020m04 (created_at);

CREATE TABLE activity.http_requests_y2020m05 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-05-01') TO ('2020-06-01');
CREATE INDEX ON activity.http_requests_y2020m05 (created_at);

CREATE TABLE activity.http_requests_y2020m06 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-06-01') TO ('2020-07-01');
CREATE INDEX ON activity.http_requests_y2020m06 (created_at);

CREATE TABLE activity.http_requests_y2020m07 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-07-01') TO ('2020-08-01');
CREATE INDEX ON activity.http_requests_y2020m07 (created_at);

CREATE TABLE activity.http_requests_y2020m08 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-08-01') TO ('2020-09-01');
CREATE INDEX ON activity.http_requests_y2020m08 (created_at);

CREATE TABLE activity.http_requests_y2020m09 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-09-01') TO ('2020-10-01');
CREATE INDEX ON activity.http_requests_y2020m09 (created_at);

CREATE TABLE activity.http_requests_y2020m10 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-10-01') TO ('2020-11-01');
CREATE INDEX ON activity.http_requests_y2020m10 (created_at);

CREATE TABLE activity.http_requests_y2020m11 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-11-01') TO ('2020-12-01');
CREATE INDEX ON activity.http_requests_y2020m11 (created_at);

CREATE TABLE activity.http_requests_y2020m12 PARTITION OF activity.http_requests
    FOR VALUES FROM ('2020-12-01') TO ('2021-01-01');
CREATE INDEX ON activity.http_requests_y2020m12 (created_at);

