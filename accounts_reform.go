package acca

// generated with gopkg.in/reform.v1

import (
	"fmt"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

type accountTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("").
func (v *accountTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("accounts").
func (v *accountTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *accountTableType) Columns() []string {
	return []string{"account_id", "customer_id", "_type", "balance", "updated_at"}
}

// NewStruct makes a new struct for that view or table.
func (v *accountTableType) NewStruct() reform.Struct {
	return new(Account)
}

// NewRecord makes a new record for that table.
func (v *accountTableType) NewRecord() reform.Record {
	return new(Account)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *accountTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// AccountTable represents accounts view or table in SQL database.
var AccountTable = &accountTableType{
	s: parse.StructInfo{Type: "Account", SQLSchema: "", SQLName: "accounts", Fields: []parse.FieldInfo{{Name: "AccountID", PKType: "int64", Column: "account_id"}, {Name: "CustomerID", PKType: "", Column: "customer_id"}, {Name: "Type", PKType: "", Column: "_type"}, {Name: "Balance", PKType: "", Column: "balance"}, {Name: "UpdatedAt", PKType: "", Column: "updated_at"}}, PKFieldIndex: 0},
	z: new(Account).Values(),
}

// String returns a string representation of this struct or record.
func (s Account) String() string {
	res := make([]string, 5)
	res[0] = "AccountID: " + reform.Inspect(s.AccountID, true)
	res[1] = "CustomerID: " + reform.Inspect(s.CustomerID, true)
	res[2] = "Type: " + reform.Inspect(s.Type, true)
	res[3] = "Balance: " + reform.Inspect(s.Balance, true)
	res[4] = "UpdatedAt: " + reform.Inspect(s.UpdatedAt, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *Account) Values() []interface{} {
	return []interface{}{
		s.AccountID,
		s.CustomerID,
		s.Type,
		s.Balance,
		s.UpdatedAt,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *Account) Pointers() []interface{} {
	return []interface{}{
		&s.AccountID,
		&s.CustomerID,
		&s.Type,
		&s.Balance,
		&s.UpdatedAt,
	}
}

// View returns View object for that struct.
func (s *Account) View() reform.View {
	return AccountTable
}

// Table returns Table object for that record.
func (s *Account) Table() reform.Table {
	return AccountTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *Account) PKValue() interface{} {
	return s.AccountID
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *Account) PKPointer() interface{} {
	return &s.AccountID
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *Account) HasPK() bool {
	return s.AccountID != AccountTable.z[AccountTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *Account) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.AccountID = int64(i64)
	} else {
		s.AccountID = pk.(int64)
	}
}

// check interfaces
var (
	_ reform.View   = AccountTable
	_ reform.Struct = new(Account)
	_ reform.Table  = AccountTable
	_ reform.Record = new(Account)
	_ fmt.Stringer  = new(Account)
)

func init() {
	parse.AssertUpToDate(&AccountTable.s, new(Account))
}
