[![CircleCI](https://circleci.com/gh/gebv/acca/tree/master.svg?style=svg)](https://circleci.com/gh/gebv/acca/tree/master)

# TODO

- [x] Rollback for failed transaction - rollback money for hold operations
- [x] Tests for basic functionality.
  - [x] CI tests.
- [ ] Basic future
  - [x] Added balance_changes + tests
  - [ ] Access to functions. For internal methods, external access is not available.
  - [ ] Prevent editing data in tables - read only. Modifing via API
  - [ ] More database-level checks
    - [ ] Transaction from one to the same account
- [ ] Added documentation
- [ ] Idempotency operation of accept\reject tx

# Future list
- [ ] Process each operation and save the executed status in current operation
- [ ] Example for analitics.
  - [ ] Beautiful graphics.
