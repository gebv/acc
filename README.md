# TODO

- [ ] Tests for basic functionality.
- [ ] Access to functions. For internal methods, external access is not available.
- [ ] Example for analitics.
  - [ ] Beautiful graphics.


## test1

```sql
INSERT INTO acca.currencies(curr) VALUES ('rub');
INSERT INTO acca.accounts(acc_id, curr, balance) VALUES('1', 'rub', 0), ('2', 'rub', 0), ('3', 'rub', 0), ('4', 'rub', 0);
SELECT acca.new_transfer('[{"src_acc_id": "1", "dst_acc_id": "2", "type": "internal", "amount": 10, "reason": "test", "meta": {}, "hold": false}, {"src_acc_id": "3", "dst_acc_id": "4", "type": "internal", "amount": 20, "reason": "test", "meta": {}, "hold": false}]', 'reason.example', '{}', false);

SELECT acca.handle_requests(1);
```

## test2

```sql
INSERT INTO acca.currencies(curr) VALUES ('rub');
INSERT INTO acca.accounts(acc_id, curr, balance) VALUES('hold1', 'rub', 0) ;
SELECT acca.new_transfer('[{"src_acc_id": "1", "dst_acc_id": "2", "type": "internal", "amount": 10, "reason": "test", "meta": {}, "hold": true, "hold_acc_id": "hold1"}, {"src_acc_id": "3", "dst_acc_id": "4", "type": "internal", "amount": 20, "reason": "test", "meta": {}, "hold": true, "hold_acc_id": "hold1"}]', 'reason.example', '{}', true);

SELECT acca.handle_requests(1);
```
