package engine

import "time"

type ProcessorCommand struct {
	TrID          int64
	CurrentStatus TransactionStatus
	NextStatus    TransactionStatus
	UpdatedAt     time.Time

	// TODO: расширить модель и добавить
	// - смена статуса для транзакции
	// - проверка закрытия всех транзакций в инвйосе и смена статуса инвойса
}
