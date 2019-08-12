package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
)

//go:generate reform

// InvoiceStatus состояние инвойса.
type InvoiceStatus string

func (s InvoiceStatus) Match(in InvoiceStatus) bool {
	return s == in
}

const (
	// DRAFT_I статус инвойса в черновике. В этом статусе позволено
	// вносить изменения в инвойс и входящие в него транзакции.
	// Только в этом статусе позволено вручную менять состав инвойса.
	DRAFT_I InvoiceStatus = "draft"

	// AUTH_I статус инвойса когда он прошел первичную валидацию и готов к дальнейшей обработке.
	AUTH_I InvoiceStatus = "auth"

	// WAIT_I статус инвойса в ожидании подтверждения чего либо. Служит для двух-этапных операций.
	WAIT_I InvoiceStatus = "wait"

	// ACCEPTED_I конечный статус инвойса. Весь инвойс принят и успешно исполнен.
	ACCEPTED_I InvoiceStatus = "accepted"

	MACCEPTED_I InvoiceStatus = "manual_accepted"

	// ACCEPTED_I конечный статус инвойса. Весь инвойс отклонен.
	REJECTED_I InvoiceStatus = "rejected"

	MREJECTED_I InvoiceStatus = "manual_rejected"
)

//reform:acca.invoices
type Invoice struct {
	// InvoiceID внутренний идентификатор инвойса.
	InvoiceID int64 `reform:"invoice_id,pk"`

	// Key внешний уникальный идентицитора инвойса.
	Key string `reform:"key"`

	// Status состояние инвойса.
	Status InvoiceStatus `reform:"status"`

	// Status куда переходит инвойс.
	NextStatus *InvoiceStatus `reform:"next_status"`

	// Strategy стратегия работы с инвойсом.
	Strategy string `reform:"strategy"`

	// Meta мета информация инвойса (учавствующая в логике).
	Meta *[]byte `reform:"meta"`

	// Payload контенйре с информацией связанной с инвойсом (не учавствтующая в логике).
	Payload *[]byte `reform:"payload"`

	// UpdatedAt дата последнего обновления инвойса (без учета входязих в нее сущеностей).
	UpdatedAt time.Time `reform:"updated_at"`

	// CreatedAt дата создания инвойса.
	CreatedAt time.Time `reform:"created_at"`
}

func (i *Invoice) BeforeInsert() error {
	i.UpdatedAt = time.Now()
	i.CreatedAt = time.Now()
	i.Status = DRAFT_I
	return nil
}

func (i *Invoice) BeforeUpdate() error {
	i.UpdatedAt = time.Now()
	return nil
}

//reform:acca.v_invoices
type ViewInvoice struct {
	// InvoiceID внутренний идентификатор инвойса.
	InvoiceID int64 `reform:"invoice_id,pk"`

	// Key внешний уникальный идентицитора инвойса.
	Key string `reform:"key"`

	Amount int64 `reform:"amount"`

	// Status состояние инвойса.
	Status InvoiceStatus `reform:"status"`

	// Status куда переходит инвойс.
	NextStatus *InvoiceStatus `reform:"next_status"`

	// Strategy стратегия работы с инвойсом.
	Strategy string `reform:"strategy"`

	// Meta мета информация инвойса (учавствующая в логике).
	Meta *[]byte `reform:"meta"`

	// Payload контенйре с информацией связанной с инвойсом (не учавствтующая в логике).
	Payload *[]byte `reform:"payload"`

	// UpdatedAt дата последнего обновления инвойса (без учета входязих в нее сущеностей).
	UpdatedAt time.Time `reform:"updated_at"`

	// CreatedAt дата создания инвойса.
	CreatedAt time.Time `reform:"created_at"`

	Transactions Transactions `reform:"transactions"`
}

type Transactions []ViewTransaction

func (t *Transactions) Scan(in interface{}) error {
	switch v := in.(type) {
	case []byte:
		buf := bytes.NewBuffer(v)
		err := json.NewDecoder(buf).Decode(t)
		return errors.Wrap(err, "Failed decode Transactions.")
	case string:
		buf := bytes.NewBufferString(v)
		err := json.NewDecoder(buf).Decode(t)
		return errors.Wrap(err, "Failed decode Transactions.")
	default:
		return fmt.Errorf(" Transactions: not expected type %T", in)
	}
}
