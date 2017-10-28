package acca

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/dialects/postgresql"
)

var db *reform.DB

func TestMain(m *testing.M) {
	var err error
	db, err = newConn(
		os.Getenv("DBADDRESS"),
		os.Getenv("DBNAME"),
		os.Getenv("DBUSER"),
		os.Getenv("DBPWD"),
	)
	if err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

func resetFixtures() {
	destroyFixtures()
	setupFixtures()
}

func setupFixtures() {
	var err error
	_, err = db.Exec(`INSERT INTO accounts(account_id, customer_id, _type, balance) VALUES
    (1, 's1000', 'system', 1000),
    (2, 'c100', 'customer', 100),
    (3, 'c10', 'customer', 10),
	(4, 'c0', 'customer', 0)`)
	log.Println("fixture: account", err)

	// _, err = db.Exec(`INSERT INTO invoices(invoice_id, order_id, destination_id, source_id, amount, created_at) VALUES
	// (1, 'o1', 1, 2, 100, now()),
	// (2, 'o2', 1, 2, 1000, now()),
	// (3, 'o3', 1, 2, 10, now())`)
	// log.Println("fixture: invoices", err)

	_, err = db.Exec(`INSERT INTO invoices(invoice_id, order_id, destination_id, amount, created_at) VALUES 
    (1, 'o1', 1, 100, now()),
    (2, 'o2', 1, 1000, now()),
	(3, 'o3', 1, 10, now())`)
	log.Println("fixture: invoices", err)
}

func destroyFixtures() {
	db.Exec(`DELETE FROM balance_changes`)
	db.Exec(`DELETE FROM transactions`)
	db.Exec(`DELETE FROM invoices`)
	db.Exec(`DELETE FROM accounts`)
}

func newConn(
	address,
	dbname,
	dbuser,
	dbpwd string,
) (db *reform.DB, err error) {
	props := url.Values{}
	props.Add("user", dbuser)
	props.Add("password", dbpwd)
	props.Add("sslmode", "disable")
	connURL := fmt.Sprintf(
		"postgres://%s/%s?%s",
		address,
		dbname,
		props.Encode(),
	)
	var conn *sql.DB
	conn, err = sql.Open(
		"postgres",
		connURL,
	)
	if err == nil {
		db = reform.NewDB(
			conn,
			postgresql.Dialect,
			reform.NewPrintfLogger(log.Printf),
		)
	}
	return
}

func dumpFromInvoice(invoiceID int64) (
	d *dump,
) {
	d = &dump{}
	d.i = &Invoice{}
	db.FindByPrimaryKeyTo(d.i, invoiceID)

	accList, _ := db.SelectAllFrom((&Account{}).View(), "")
	for _, item := range accList {
		d.accs = append(d.accs, item.(*Account))
	}

	txList, _ := db.SelectAllFrom((&Transaction{}).View(), "WHERE invoice_id = $1", invoiceID)
	for _, item := range txList {
		tx := item.(*Transaction)
		d.txs = append(d.txs, tx)

		chList, _ := db.SelectAllFrom((&BalanceChanges{}).View(), "WHERE transaction_id = $1", tx.TransactionID)
		for _, item := range chList {
			d.bcs = append(d.bcs, item.(*BalanceChanges))
		}
	}

	return
}

type dump struct {
	i    *Invoice
	accs []*Account
	bcs  []*BalanceChanges
	txs  []*Transaction
}

func (d *dump) FindAccount(objID int64) *Account {
	for _, item := range d.accs {
		if item.AccountID == objID {
			return item
		}
	}
	return nil
}

func (d *dump) FindTx(objID int64) *Transaction {
	for _, item := range d.txs {
		if item.TransactionID == objID {
			return item
		}
	}
	return nil
}

func (d *dump) ChangesByAcc(objID int64) (res []*BalanceChanges) {
	for _, item := range d.bcs {
		if item.AccountID == objID {
			res = append(res, item)
		}
	}
	return
}
