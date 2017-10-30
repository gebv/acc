package acca

// generated with gopkg.in/reform.v1

import (
	"fmt"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

type transactionTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("finances").
func (v *transactionTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("transactions").
func (v *transactionTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *transactionTableType) Columns() []string {
	return []string{"transaction_id", "invoice_id", "amount", "source", "destination", "status", "created_at", "closed_at"}
}

// NewStruct makes a new struct for that view or table.
func (v *transactionTableType) NewStruct() reform.Struct {
	return new(Transaction)
}

// NewRecord makes a new record for that table.
func (v *transactionTableType) NewRecord() reform.Record {
	return new(Transaction)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *transactionTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// TransactionTable represents transactions view or table in SQL database.
var TransactionTable = &transactionTableType{
	s: parse.StructInfo{Type: "Transaction", SQLSchema: "finances", SQLName: "transactions", Fields: []parse.FieldInfo{{Name: "TransactionID", PKType: "int64", Column: "transaction_id"}, {Name: "InvoiceID", PKType: "", Column: "invoice_id"}, {Name: "Amount", PKType: "", Column: "amount"}, {Name: "Source", PKType: "", Column: "source"}, {Name: "Destination", PKType: "", Column: "destination"}, {Name: "Status", PKType: "", Column: "status"}, {Name: "CreatedAt", PKType: "", Column: "created_at"}, {Name: "ClosedAt", PKType: "", Column: "closed_at"}}, PKFieldIndex: 0},
	z: new(Transaction).Values(),
}

// String returns a string representation of this struct or record.
func (s Transaction) String() string {
	res := make([]string, 8)
	res[0] = "TransactionID: " + reform.Inspect(s.TransactionID, true)
	res[1] = "InvoiceID: " + reform.Inspect(s.InvoiceID, true)
	res[2] = "Amount: " + reform.Inspect(s.Amount, true)
	res[3] = "Source: " + reform.Inspect(s.Source, true)
	res[4] = "Destination: " + reform.Inspect(s.Destination, true)
	res[5] = "Status: " + reform.Inspect(s.Status, true)
	res[6] = "CreatedAt: " + reform.Inspect(s.CreatedAt, true)
	res[7] = "ClosedAt: " + reform.Inspect(s.ClosedAt, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *Transaction) Values() []interface{} {
	return []interface{}{
		s.TransactionID,
		s.InvoiceID,
		s.Amount,
		s.Source,
		s.Destination,
		s.Status,
		s.CreatedAt,
		s.ClosedAt,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *Transaction) Pointers() []interface{} {
	return []interface{}{
		&s.TransactionID,
		&s.InvoiceID,
		&s.Amount,
		&s.Source,
		&s.Destination,
		&s.Status,
		&s.CreatedAt,
		&s.ClosedAt,
	}
}

// View returns View object for that struct.
func (s *Transaction) View() reform.View {
	return TransactionTable
}

// Table returns Table object for that record.
func (s *Transaction) Table() reform.Table {
	return TransactionTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *Transaction) PKValue() interface{} {
	return s.TransactionID
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *Transaction) PKPointer() interface{} {
	return &s.TransactionID
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *Transaction) HasPK() bool {
	return s.TransactionID != TransactionTable.z[TransactionTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *Transaction) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.TransactionID = int64(i64)
	} else {
		s.TransactionID = pk.(int64)
	}
}

// check interfaces
var (
	_ reform.View   = TransactionTable
	_ reform.Struct = new(Transaction)
	_ reform.Table  = TransactionTable
	_ reform.Record = new(Transaction)
	_ fmt.Stringer  = new(Transaction)
)

type balanceChangesTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("finances").
func (v *balanceChangesTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("balance_changes").
func (v *balanceChangesTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *balanceChangesTableType) Columns() []string {
	return []string{"change_id", "account_id", "transaction_id", "tx_type", "amount", "balance", "created_at"}
}

// NewStruct makes a new struct for that view or table.
func (v *balanceChangesTableType) NewStruct() reform.Struct {
	return new(BalanceChanges)
}

// NewRecord makes a new record for that table.
func (v *balanceChangesTableType) NewRecord() reform.Record {
	return new(BalanceChanges)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *balanceChangesTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// BalanceChangesTable represents balance_changes view or table in SQL database.
var BalanceChangesTable = &balanceChangesTableType{
	s: parse.StructInfo{Type: "BalanceChanges", SQLSchema: "finances", SQLName: "balance_changes", Fields: []parse.FieldInfo{{Name: "ChangeID", PKType: "int64", Column: "change_id"}, {Name: "AccountID", PKType: "", Column: "account_id"}, {Name: "TransactionID", PKType: "", Column: "transaction_id"}, {Name: "Type", PKType: "", Column: "tx_type"}, {Name: "Amount", PKType: "", Column: "amount"}, {Name: "Balance", PKType: "", Column: "balance"}, {Name: "CreatedAt", PKType: "", Column: "created_at"}}, PKFieldIndex: 0},
	z: new(BalanceChanges).Values(),
}

// String returns a string representation of this struct or record.
func (s BalanceChanges) String() string {
	res := make([]string, 7)
	res[0] = "ChangeID: " + reform.Inspect(s.ChangeID, true)
	res[1] = "AccountID: " + reform.Inspect(s.AccountID, true)
	res[2] = "TransactionID: " + reform.Inspect(s.TransactionID, true)
	res[3] = "Type: " + reform.Inspect(s.Type, true)
	res[4] = "Amount: " + reform.Inspect(s.Amount, true)
	res[5] = "Balance: " + reform.Inspect(s.Balance, true)
	res[6] = "CreatedAt: " + reform.Inspect(s.CreatedAt, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *BalanceChanges) Values() []interface{} {
	return []interface{}{
		s.ChangeID,
		s.AccountID,
		s.TransactionID,
		s.Type,
		s.Amount,
		s.Balance,
		s.CreatedAt,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *BalanceChanges) Pointers() []interface{} {
	return []interface{}{
		&s.ChangeID,
		&s.AccountID,
		&s.TransactionID,
		&s.Type,
		&s.Amount,
		&s.Balance,
		&s.CreatedAt,
	}
}

// View returns View object for that struct.
func (s *BalanceChanges) View() reform.View {
	return BalanceChangesTable
}

// Table returns Table object for that record.
func (s *BalanceChanges) Table() reform.Table {
	return BalanceChangesTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *BalanceChanges) PKValue() interface{} {
	return s.ChangeID
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *BalanceChanges) PKPointer() interface{} {
	return &s.ChangeID
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *BalanceChanges) HasPK() bool {
	return s.ChangeID != BalanceChangesTable.z[BalanceChangesTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *BalanceChanges) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.ChangeID = int64(i64)
	} else {
		s.ChangeID = pk.(int64)
	}
}

// check interfaces
var (
	_ reform.View   = BalanceChangesTable
	_ reform.Struct = new(BalanceChanges)
	_ reform.Table  = BalanceChangesTable
	_ reform.Record = new(BalanceChanges)
	_ fmt.Stringer  = new(BalanceChanges)
)

func init() {
	parse.AssertUpToDate(&TransactionTable.s, new(Transaction))
	parse.AssertUpToDate(&BalanceChangesTable.s, new(BalanceChanges))
}
