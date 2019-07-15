package strategies

import (
	"fmt"
	"testing"

	"github.com/gebv/acca/engine"
)

func TestStrategy(t *testing.T) {
	ptrTrStatus := func(s engine.TransactionStatus) *engine.TransactionStatus { return &s }
	ptrTrStatusToStr := func(s *engine.TransactionStatus) string {
		if s != nil {
			return string(*s)
		}
		return ""
	}
	ptrInvStatus := func(s engine.InvoiceStatus) *engine.InvoiceStatus { return &s }
	ptrInvStatusToStr := func(s *engine.InvoiceStatus) string {
		if s != nil {
			return string(*s)
		}
		return ""
	}

	if S == nil {
		t.Error("S is nil ")
	}
	tests := []struct {
		invStatus     *engine.InvoiceStatus
		trStatus      *engine.TransactionStatus
		hold          bool
		wantInvStatus engine.InvoiceStatus
		wantTrStatus  engine.TransactionStatus
		wantErr       bool
	}{
		{
			invStatus:     nil,
			trStatus:      ptrTrStatus(engine.AUTH_TX),
			hold:          false,
			wantInvStatus: engine.AUTH_I,
			wantTrStatus:  engine.AUTH_TX,
			wantErr:       false,
		},
		{
			invStatus:     ptrInvStatus(engine.ACCEPTED_I),
			trStatus:      nil,
			hold:          false,
			wantInvStatus: engine.ACCEPTED_I,
			wantTrStatus:  engine.ACCEPTED_TX,
			wantErr:       false,
		},
		{
			invStatus:     ptrInvStatus(engine.REJECTED_I),
			trStatus:      nil,
			hold:          false,
			wantInvStatus: engine.ACCEPTED_I,
			wantTrStatus:  engine.ACCEPTED_TX,
			wantErr:       true,
		},
		{
			invStatus:     nil,
			trStatus:      ptrTrStatus(engine.REJECTED_TX),
			hold:          false,
			wantInvStatus: engine.ACCEPTED_I,
			wantTrStatus:  engine.ACCEPTED_TX,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		tname := fmt.Sprintf("invStatus: %s trStatus: %s setInvStatus: %s setTrStatus: %s wantInvStatus: %s wanTrStatus: %s wantErr: %v",
			noDbFromTest.inv.Status,
			noDbFromTest.tr.Status,
			ptrInvStatusToStr(tt.invStatus),
			ptrTrStatusToStr(tt.trStatus),
			tt.wantInvStatus,
			tt.wantTrStatus,
			tt.wantErr,
		)
		t.Run(tname, func(t *testing.T) {
			if tt.invStatus != nil {
				if err := SetInvoiceStatus(noDbFromTest.inv.InvoiceID, *tt.invStatus); (err != nil) != tt.wantErr {
					t.Errorf("SetInvoiceStatus() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if tt.trStatus != nil {
				if err := SetTransactionStatus(noDbFromTest.tr.TransactionID, *tt.trStatus); (err != nil) != tt.wantErr {
					t.Errorf("SetTransactionStatus() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else {
				t.Error("Set status is nil")
			}
			if noDbFromTest.inv.Status != tt.wantInvStatus {
				t.Errorf("Invoice status = %v, wantStatus %v", noDbFromTest.inv.Status, tt.wantInvStatus)
			}
			if noDbFromTest.tr.Status != tt.wantTrStatus {
				t.Errorf("Transaction status = %v, wantStatus %v", noDbFromTest.tr.Status, tt.wantTrStatus)
			}
		})
	}
}
