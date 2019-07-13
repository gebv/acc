package engine

import "time"

type processorCommand struct {
	txID          int64
	currentStatus TransactionStatus
	nextStatus    TransactionStatus
	updatedAt     time.Time

	// TODO: расширить модель и добавить
	// - смена статуса для транзакции
	// - проверка закрытия всех транзакций в инвйосе и смена статуса инвойса
}
