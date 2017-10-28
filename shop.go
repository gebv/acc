package acca

type Shop interface {
	Invoice(orderID string, amount int64) (*Invoice, error)
	Pay(invoiceID, sourceID int64) error
}

type ShopInspector interface {
	CanPay(invoiceID, sourceID int64) error
}

// type ShopExample struct {
// 	c Cashier
//  s Stock
// }

// func (s *ShopExample) NewOrder() (Order, error) {
// 	// СОхранить заказ
// 	return nil, nil
// }

// // Invoice Выставить счет на оплату.
// func (s *ShopExample) Invoice(orderID string, amount int64) (*Invoice, error) {
// 	o, _ := s.s.FindOrderByID(orderID)
// 	i := &Invoice{
// 		OrderID:       o.OrderID(),
// 		DestinationID: o.DestinationID(),
// 		Amount:        amount,
// 	}

// 	return i, nil
// }

// // Pay оплатить счет.
// func (s *ShopExample) Pay(invoiceID, sourceID string) error {
// 	return nil
// }
