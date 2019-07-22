// Code generated by gopkg.in/reform.v1. DO NOT EDIT.

package provider

import (
	"fmt"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

type invoiceTransactionsExtOrdersTableType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("").
func (v *invoiceTransactionsExtOrdersTableType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("invoice_transactions_ext_orders").
func (v *invoiceTransactionsExtOrdersTableType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *invoiceTransactionsExtOrdersTableType) Columns() []string {
	return []string{"order_number", "payment_system_name", "raw_order_status", "order_status", "created_at", "updated_at", "ext_updated_at"}
}

// NewStruct makes a new struct for that view or table.
func (v *invoiceTransactionsExtOrdersTableType) NewStruct() reform.Struct {
	return new(InvoiceTransactionsExtOrders)
}

// NewRecord makes a new record for that table.
func (v *invoiceTransactionsExtOrdersTableType) NewRecord() reform.Record {
	return new(InvoiceTransactionsExtOrders)
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *invoiceTransactionsExtOrdersTableType) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

// InvoiceTransactionsExtOrdersTable represents invoice_transactions_ext_orders view or table in SQL database.
var InvoiceTransactionsExtOrdersTable = &invoiceTransactionsExtOrdersTableType{
	s: parse.StructInfo{Type: "InvoiceTransactionsExtOrders", SQLSchema: "", SQLName: "invoice_transactions_ext_orders", Fields: []parse.FieldInfo{{Name: "OrderNumber", Type: "string", Column: "order_number"}, {Name: "PaymentSystemName", Type: "string", Column: "payment_system_name"}, {Name: "RawOrderStatus", Type: "string", Column: "raw_order_status"}, {Name: "OrderStatus", Type: "string", Column: "order_status"}, {Name: "CreatedAt", Type: "time.Time", Column: "created_at"}, {Name: "UpdatedAt", Type: "time.Time", Column: "updated_at"}, {Name: "ExtUpdatedAt", Type: "time.Time", Column: "ext_updated_at"}}, PKFieldIndex: 0},
	z: new(InvoiceTransactionsExtOrders).Values(),
}

// String returns a string representation of this struct or record.
func (s InvoiceTransactionsExtOrders) String() string {
	res := make([]string, 7)
	res[0] = "OrderNumber: " + reform.Inspect(s.OrderNumber, true)
	res[1] = "PaymentSystemName: " + reform.Inspect(s.PaymentSystemName, true)
	res[2] = "RawOrderStatus: " + reform.Inspect(s.RawOrderStatus, true)
	res[3] = "OrderStatus: " + reform.Inspect(s.OrderStatus, true)
	res[4] = "CreatedAt: " + reform.Inspect(s.CreatedAt, true)
	res[5] = "UpdatedAt: " + reform.Inspect(s.UpdatedAt, true)
	res[6] = "ExtUpdatedAt: " + reform.Inspect(s.ExtUpdatedAt, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *InvoiceTransactionsExtOrders) Values() []interface{} {
	return []interface{}{
		s.OrderNumber,
		s.PaymentSystemName,
		s.RawOrderStatus,
		s.OrderStatus,
		s.CreatedAt,
		s.UpdatedAt,
		s.ExtUpdatedAt,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *InvoiceTransactionsExtOrders) Pointers() []interface{} {
	return []interface{}{
		&s.OrderNumber,
		&s.PaymentSystemName,
		&s.RawOrderStatus,
		&s.OrderStatus,
		&s.CreatedAt,
		&s.UpdatedAt,
		&s.ExtUpdatedAt,
	}
}

// View returns View object for that struct.
func (s *InvoiceTransactionsExtOrders) View() reform.View {
	return InvoiceTransactionsExtOrdersTable
}

// Table returns Table object for that record.
func (s *InvoiceTransactionsExtOrders) Table() reform.Table {
	return InvoiceTransactionsExtOrdersTable
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *InvoiceTransactionsExtOrders) PKValue() interface{} {
	return s.OrderNumber
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *InvoiceTransactionsExtOrders) PKPointer() interface{} {
	return &s.OrderNumber
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *InvoiceTransactionsExtOrders) HasPK() bool {
	return s.OrderNumber != InvoiceTransactionsExtOrdersTable.z[InvoiceTransactionsExtOrdersTable.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *InvoiceTransactionsExtOrders) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.OrderNumber = string(i64)
	} else {
		s.OrderNumber = pk.(string)
	}
}

// check interfaces
var (
	_ reform.View   = InvoiceTransactionsExtOrdersTable
	_ reform.Struct = (*InvoiceTransactionsExtOrders)(nil)
	_ reform.Table  = InvoiceTransactionsExtOrdersTable
	_ reform.Record = (*InvoiceTransactionsExtOrders)(nil)
	_ fmt.Stringer  = (*InvoiceTransactionsExtOrders)(nil)
)

func init() {
	parse.AssertUpToDate(&InvoiceTransactionsExtOrdersTable.s, new(InvoiceTransactionsExtOrders))
}
