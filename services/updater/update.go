package updater

import (
	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/services/invoices"
)

type Update struct {
	UpdatedInvoice     *UpdatedInvoice     `json:"invoice"`
	UpdatedTransaction *UpdatedTransaction `json:"transaction"`
}

type UpdatedInvoice struct {
	InvoiceID int64 `json:"invoice_id"`
	Status    engine.InvoiceStatus
}

type UpdatedTransaction struct {
	TransactionID int64 `json:"transaction_id"`
	Status        engine.TransactionStatus
}

func convertUpdate(m *Update) *api.Update {
	var u *api.Update
	if m.UpdatedInvoice != nil {
		u = &api.Update{
			Type: &api.Update_UpdatedInvoice{
				UpdatedInvoice: &api.UpdatedInvoice{
					InvoiceId: m.UpdatedInvoice.InvoiceID,
					Status:    invoices.MapInvStatusToApiInvStatus[m.UpdatedInvoice.Status],
				},
			},
		}
	} else if m.UpdatedTransaction != nil {
		u = &api.Update{
			Type: &api.Update_UpdatedTransaction{
				UpdatedTransaction: &api.UpdatedTransaction{
					TransactionId: m.UpdatedTransaction.TransactionID,
					Status:        invoices.MapTrStatusToApiTrStatus[m.UpdatedTransaction.Status],
				},
			},
		}
	}
	return u
}
