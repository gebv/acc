# accounting
Financial accounting

See database schema https://github.com/gebv/acca/issues/7

# quick start

[schema database](schema.sql)

``` shell
docker-compose up -d
# setup schema
make test
```

for reference see tests 
* [cashier](cashier_pg_test.go)

# overview

## cashier

low-level layer responsible for the transfer of funds between accounts

```golang
type Cashier interface {
	// Hold first phase of payment - hold amount of invoice.
	Hold(sourceID, invoiceID int64) (txID int64, err error)

	// Accept second phase of payment - payment confimration.
	Accept(txID int64) (err error)

	// Reject second phase of payment - payment not rejected.
	Reject(txID int64) (err error)
}

```

## shop

```golang

type Shop interface {
	// Invocie returns the invoice for payment.
	Invoice(orderID string, amount int64) (*Invoice, error)

	// Pay payment of invoice.
	Pay(invoiceID, sourceID int64) error
}
```

### Settings VSCode workspace

```json
{
    "go.testEnvFile": "${workspaceRoot}/.env"
}
```
