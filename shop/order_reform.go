package shop

// generated with gopkg.in/reform.v1

import (
	"fmt"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)

type orderViewType struct {
	s parse.StructInfo
	z []interface{}
}

// Schema returns a schema name in SQL database ("shop").
func (v *orderViewType) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("orders").
func (v *orderViewType) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *orderViewType) Columns() []string {
	return []string{"order_id", "destination_id", "order_type", "total", "closed", "created_at"}
}

// NewStruct makes a new struct for that view or table.
func (v *orderViewType) NewStruct() reform.Struct {
	return new(order)
}

// orderView represents orders view or table in SQL database.
var orderView = &orderViewType{
	s: parse.StructInfo{Type: "order", SQLSchema: "shop", SQLName: "orders", Fields: []parse.FieldInfo{{Name: "OrderID", PKType: "", Column: "order_id"}, {Name: "DestinationID", PKType: "", Column: "destination_id"}, {Name: "Type", PKType: "", Column: "order_type"}, {Name: "Total", PKType: "", Column: "total"}, {Name: "Closed", PKType: "", Column: "closed"}, {Name: "CreatedAt", PKType: "", Column: "created_at"}}, PKFieldIndex: -1},
	z: new(order).Values(),
}

// String returns a string representation of this struct or record.
func (s order) String() string {
	res := make([]string, 6)
	res[0] = "OrderID: " + reform.Inspect(s.OrderID, true)
	res[1] = "DestinationID: " + reform.Inspect(s.DestinationID, true)
	res[2] = "Type: " + reform.Inspect(s.Type, true)
	res[3] = "Total: " + reform.Inspect(s.Total, true)
	res[4] = "Closed: " + reform.Inspect(s.Closed, true)
	res[5] = "CreatedAt: " + reform.Inspect(s.CreatedAt, true)
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *order) Values() []interface{} {
	return []interface{}{
		s.OrderID,
		s.DestinationID,
		s.Type,
		s.Total,
		s.Closed,
		s.CreatedAt,
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *order) Pointers() []interface{} {
	return []interface{}{
		&s.OrderID,
		&s.DestinationID,
		&s.Type,
		&s.Total,
		&s.Closed,
		&s.CreatedAt,
	}
}

// View returns View object for that struct.
func (s *order) View() reform.View {
	return orderView
}

// check interfaces
var (
	_ reform.View   = orderView
	_ reform.Struct = new(order)
	_ fmt.Stringer  = new(order)
)

func init() {
	parse.AssertUpToDate(&orderView.s, new(order))
}
