
-- transfer
-- creates a request to execute a transaction
CREATE OR REPLACE FUNCTION acca.new_transfer(
    _operations jsonb,
    _reason ltree,
    _meta jsonb,
    OUT _tx_id bigint
) RETURNS bigint AS $$
    BEGIN
        /*
        - создать транзакцию
        - создать запрос на авторизацию транзакции
        */
        -- new transaction
        INSERT INTO acca.transactions (tx_id, reason, meta, status) VALUES (DEFAULT, _reason, _meta, 'draft'::acca.transaction_status) RETURNING acca.transactions.tx_id INTO _tx_id;

        -- authorization transaction
        INSERT INTO acca.requests_queue(tx_id, type) VALUES(_tx_id, 'auth'::acca.request_type);

        -- processing incoming operations
        INSERT INTO acca.operations(tx_id, src_acc_id, dst_acc_id, type, amount, reason, meta, hold, hold_acc_id, status)
        SELECT
            (SELECT _tx_id) AS tx_id,
            src_acc_id,
            dst_acc_id,
            type,
            amount,
            reason,
            meta,
            hold,
            hold_acc_id,
            'draft'::acca.operation_status
        FROM jsonb_populate_recordset(null::acca.operations, _operations);

        PERFORM pg_notify('new_transaction', json_build_object('tx_id', _tx_id)::text);
    END;
$$ language plpgsql;

-- accept_tx
-- create request to accept a transaction
CREATE OR REPLACE FUNCTION acca.accept_tx(
    _tx_id bigint
) RETURNS void AS $$
    DECLARE
        _tx_status acca.transaction_status;
    BEGIN
        SELECT status INTO _tx_status FROM acca.transactions WHERE tx_id = _tx_id;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'Transaction not found';
        END IF;

        IF _tx_status <> 'auth' THEN
            RAISE EXCEPTION 'Transaction has already closed (or not found): status=%', _tx_status::text;
        END IF;

        INSERT INTO acca.requests_queue(tx_id, type) VALUES(_tx_id, 'accept'::acca.request_type);
    END;
$$ language plpgsql;

-- rollback_tx
CREATE OR REPLACE FUNCTION acca.rollback_tx(
    _tx_id bigint
) RETURNS void AS $$
    DECLARE
        _tx_status acca.transaction_status;
    BEGIN
        SELECT status INTO _tx_status FROM acca.transactions WHERE tx_id = _tx_id;

        IF NOT FOUND THEN
            RAISE EXCEPTION 'Transaction not found';
        END IF;

        IF _tx_status <> 'failed' THEN
            RAISE EXCEPTION 'Not allowed status of the transaction for rollback: status=%', _tx_status::text;
        END IF;

        INSERT INTO acca.requests_queue(tx_id, type) VALUES(_tx_id, 'rollback'::acca.request_type);
    END;
$$ language plpgsql;

-- create request to reject a transaction
CREATE OR REPLACE FUNCTION acca.reject_tx(
    _tx_id bigint
) RETURNS void AS $$
    DECLARE
        _tx_status acca.transaction_status;
    BEGIN
        SELECT status INTO _tx_status FROM acca.transactions WHERE tx_id = _tx_id;
        IF _tx_status <> 'auth' THEN
            RAISE EXCEPTION 'Transaction has already closed (or not found): status=%', _tx_status::text;
        END IF;

        INSERT INTO acca.requests_queue(tx_id, type) VALUES(_tx_id, 'reject'::acca.request_type);
    END;
$$ language plpgsql;

-- auth_operation
-- handler new operation
-- internal method
CREATE OR REPLACE FUNCTION acca.auth_operation(
    _oper_id bigint
) RETURNS void AS $$
    DECLARE
        _amount numeric(69, 0);
        _src_acc_id bigint;
        _dst_acc_id bigint;
        _hold_acc_id bigint;
        _type acca.operation_type;
        _hold boolean;
        _oper_next_status acca.operation_status;
        _tx_id bigint;

        __current_acc_id bigint;
    BEGIN
        SELECT
            tx_id,
            amount,
            src_acc_id,
            dst_acc_id,
            hold,
            hold_acc_id,
            type
        INTO _tx_id, _amount, _src_acc_id, _dst_acc_id, _hold, _hold_acc_id, _type
        FROM acca.operations WHERE oper_id = _oper_id;

        BEGIN
            CASE _type
                WHEN 'internal' THEN
                    __current_acc_id := _src_acc_id;

                    IF _hold THEN
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;
                    ELSE
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                    END IF;
                WHEN 'recharge' THEN
                    IF _hold THEN
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;
                    ELSE
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                    END IF;
                WHEN 'withdraw' THEN
                    IF _hold THEN
                        __current_acc_id := _src_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;

                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;
                    ELSE
                        __current_acc_id := _src_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;

                        __current_acc_id := _dst_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                    END IF;
                ELSE
                    RAISE EXCEPTION 'Unexpected operation type: oper_id=%, type=%', _oper_id, _type::text;
            END CASE;
        EXCEPTION
            WHEN others THEN
                RAISE EXCEPTION 'Failed handler operation: oper_id=%, acc_id=%, errm=%.', _oper_id, __current_acc_id, SQLERRM;
        END;

        -- update status for operation
        IF _hold THEN
            _oper_next_status = 'hold';
        ELSE
            _oper_next_status = 'accepted';
        END IF;
        UPDATE acca.operations SET status = _oper_next_status WHERE oper_id = _oper_id;

        PERFORM pg_notify('oper_update_status', json_build_object('oper_id', _oper_id, 'src_acc_id', _src_acc_id, 'dst_acc_id', _dst_acc_id, 'new_status', _oper_next_status, 'amount', _amount, 'type', _type, 'tx_id', _tx_id)::text);
    END;
$$ language plpgsql;

-- accept_operation
CREATE OR REPLACE FUNCTION acca.accept_operation(
    _oper_id bigint
) RETURNS void AS $$
    DECLARE
        _amount numeric(69, 0);
        _src_acc_id bigint;
        _dst_acc_id bigint;
        _hold_acc_id bigint;
        _type acca.operation_type;
        _hold boolean;
        _oper_next_status acca.operation_status;
        _tx_id bigint;

        __current_acc_id bigint;
    BEGIN
        SELECT
            tx_id,
            amount,
            src_acc_id,
            dst_acc_id,
            hold,
            hold_acc_id,
            type
        INTO _tx_id, _amount, _src_acc_id, _dst_acc_id, _hold, _hold_acc_id, _type
        FROM acca.operations WHERE oper_id = _oper_id;

        BEGIN
            CASE _type
                WHEN 'internal' THEN
                    __current_acc_id := _hold_acc_id;

                    IF _hold THEN
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                    END IF;
                WHEN 'recharge' THEN
                    IF _hold THEN
                        __current_acc_id := _hold_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                    END IF;
                WHEN 'withdraw' THEN
                    IF _hold THEN
                        __current_acc_id := _hold_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;

                        __current_acc_id := _dst_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                    END IF;
                ELSE
                    RAISE EXCEPTION 'Unexpected operation type: oper_id=%, type=%', _oper_id, _type::text;
            END CASE;
        EXCEPTION
            WHEN others THEN
                RAISE EXCEPTION 'Failed handler operation: oper_id=%, acc_id=%, errm=%.', _oper_id, __current_acc_id, SQLERRM;
        END;

        -- update status for operation
        IF _hold THEN
            _oper_next_status = 'accepted';
        END IF;
        UPDATE acca.operations SET status = _oper_next_status WHERE oper_id = _oper_id;

        PERFORM pg_notify('oper_update_status', json_build_object('oper_id', _oper_id, 'src_acc_id', _src_acc_id, 'dst_acc_id', _dst_acc_id, 'new_status', _oper_next_status, 'amount', _amount, 'type', _type, 'tx_id', _tx_id)::text);
    END;
$$ language plpgsql;

-- reject_operation
CREATE OR REPLACE FUNCTION acca.reject_operation(
    _oper_id bigint
) RETURNS void AS $$
    DECLARE
        _amount numeric(69, 0);
        _src_acc_id bigint;
        _dst_acc_id bigint;
        _hold_acc_id bigint;
        _type acca.operation_type;
        _hold boolean;
        _oper_next_status acca.operation_status;
        _tx_id bigint;

        __current_acc_id bigint;
    BEGIN
        SELECT
            tx_id,
            amount,
            src_acc_id,
            dst_acc_id,
            hold,
            hold_acc_id,
            type
        INTO _tx_id, _amount, _src_acc_id, _dst_acc_id, _hold, _hold_acc_id, _type
        FROM acca.operations WHERE oper_id = _oper_id;

        BEGIN
            CASE _type
                WHEN 'internal' THEN
                    __current_acc_id := _hold_acc_id;

                    IF _hold THEN
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                    ELSE
                        __current_acc_id := _dst_acc_id;

                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                    END IF;
                WHEN 'recharge' THEN
                    IF _hold THEN
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;
                    ELSE
                        __current_acc_id := _src_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;

                        __current_acc_id := _dst_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _dst_acc_id;
                    END IF;
                WHEN 'withdraw' THEN
                    IF _hold THEN
                        __current_acc_id := _hold_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;

                        __current_acc_id := _src_acc_id;
                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                    ELSE
                        __current_acc_id := _hold_acc_id;
                        UPDATE acca.accounts SET balance = balance - _amount, last_oper_id = _oper_id WHERE acc_id = _hold_acc_id;

                        UPDATE acca.accounts SET balance = balance + _amount, last_oper_id = _oper_id WHERE acc_id = _src_acc_id;
                    END IF;
                ELSE
                    RAISE EXCEPTION 'Unexpected operation type: oper_id=%, type=%', _oper_id, _type::text;
            END CASE;
        EXCEPTION
            WHEN others THEN
                RAISE EXCEPTION 'Failed handler operation: oper_id=%, acc_id=%, errm=%.', _oper_id, __current_acc_id, SQLERRM;
        END;

        -- update status for operation
        _oper_next_status = 'rejected';
        UPDATE acca.operations SET status = _oper_next_status WHERE oper_id = _oper_id;

        PERFORM pg_notify('oper_update_status', json_build_object('oper_id', _oper_id, 'src_acc_id', _src_acc_id, 'dst_acc_id', _dst_acc_id, 'new_status', _oper_next_status, 'amount', _amount, 'type', _type, 'tx_id', _tx_id)::text);
    END;
$$ language plpgsql;

CREATE OR REPLACE FUNCTION acca.update_status_transaction(
    _tx_id bigint,
    OUT _tx_new_status acca.transaction_status
) RETURNS acca.transaction_status AS $$
    DECLARE
        num_total integer;
        num_hold integer;
        num_accepted integer;
        num_rejected integer;
        num_draft integer;
    BEGIN
        -- TODO: refactoring?
        SELECT count(*) INTO num_total FROM acca.operations WHERE tx_id = _tx_id;
        -- SELECT count(*) INTO num_hold FROM acca.operations WHERE tx_id = _tx_id AND status = 'hold';
        SELECT count(*) INTO num_accepted FROM acca.operations WHERE tx_id = _tx_id AND status = 'accepted';
        SELECT count(*) INTO num_rejected FROM acca.operations WHERE tx_id = _tx_id AND status = 'rejected';
        -- SELECT count(*) INTO num_draft FROM acca.operations WHERE tx_id = _tx_id AND status = 'draft';

        IF num_total = num_accepted THEN
            _tx_new_status := 'accepted'::acca.transaction_status;

        ELSIF num_total = num_rejected THEN
            _tx_new_status := 'rejected'::acca.transaction_status;
        ELSE
            _tx_new_status := 'auth'::acca.transaction_status;
        END IF;

        UPDATE acca.transactions
            SET status = _tx_new_status
            WHERE tx_id = _tx_id;
    END;
$$ language plpgsql;

-- handler requests from queue
CREATE OR REPLACE FUNCTION acca.handle_requests(
    _limit bigint
) RETURNS void AS $$
    declare
        reqrow record;
        failed boolean;
        failed_errm text;

        _tx_new_status acca.transaction_status;
        -- num_tx_opers bigint := 1;
    BEGIN
        FOR reqrow IN
            SELECT tx_id, type FROM acca.requests_queue ORDER BY created_at ASC LIMIT _limit
        LOOP
            -- required remove from queue
            INSERT INTO acca.requests_history(tx_id, type, created_at, executed_at) SELECT tx_id, type, created_at, now() FROM acca.requests_queue WHERE tx_id = reqrow.tx_id;
            DELETE FROM acca.requests_queue WHERE tx_id = reqrow.tx_id AND type = reqrow.type;

            BEGIN
                CASE reqrow.type
                    WHEN 'auth' THEN
                        PERFORM acca.auth_operation(oper_id) FROM acca.operations WHERE tx_id = reqrow.tx_id AND status = 'draft';
                    WHEN 'accept' THEN
                        PERFORM acca.accept_operation(oper_id) FROM acca.operations WHERE tx_id = reqrow.tx_id AND status = 'hold';
                    WHEN 'reject' THEN
                        PERFORM acca.reject_operation(oper_id) FROM acca.operations WHERE tx_id = reqrow.tx_id AND status = 'hold';
                    WHEN 'rollback' THEN
                        PERFORM acca.reject_operation(oper_id) FROM acca.operations WHERE tx_id = reqrow.tx_id AND status = 'hold';
                        PERFORM acca.reject_operation(oper_id) FROM acca.operations WHERE tx_id = reqrow.tx_id AND status = 'accepted';
                    ELSE
                        RAISE EXCEPTION 'Unexpected request type: tx_id=%, type=%.', reqrow.tx_id, reqrow.type::text;
                END CASE;

            EXCEPTION
                WHEN others THEN
                    -- do nothig
                    failed := true;
                    failed_errm := SQLERRM;
            END;

            IF failed THEN
                -- TODO: add error message to transaction

                _tx_new_status := 'failed'::acca.transaction_status;

                UPDATE acca.transactions
                    SET status = _tx_new_status,
                    errm = failed_errm
                    WHERE tx_id = reqrow.tx_id;
            ELSE
                -- upd status for tx
                SELECT acca.update_status_transaction(reqrow.tx_id) INTO _tx_new_status;
            END IF;

            PERFORM pg_notify('tx_update_status', json_build_object('tx_id', reqrow.tx_id, 'new_status', _tx_new_status)::text);

            failed := false;
            failed_errm := '';
        END LOOP;
    END;
$$ language plpgsql;
