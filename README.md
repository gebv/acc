# accounting
Financial accounting

* accounts - информация о счете и связанной с ней вспомогательной информацией (валюта, владелец, тип счета и тп)
* balance_changes - состояние счета во времени
* transactions - транзакции

amounts
* id serial
* info (currency, etc)
* amount numeric

transactions
* id serial
* amount numeric
* status enum
* balance numeric
* amount_id => amounts
* source => accounts
* destination => accounts

balance_changes
* id serial
* amount numeric
* type enum
* transaction_id => transactions
* balance numeric
* amount_id => amounts

API
* transfer_money(source, destination, amount, hold) transaction_id - перевод со счета на счет. Создается транзакция со статусом authorization, проверка source_amount и destination_amount на возможность обеспечения, запись в balance_changes для source.Для однофакторных (внутренних например) транзакций hold=false, в случае многофакторной транзакции (например связь с внешним миром) hold=true - средства замораживаются (становятся не доступными ни кому) пока либо не accept_transaction или cancel_transaction. В случае однофакторной транзакции запись в balance_changes для destination, транзакция состатусом accepted. В противном случае error (не достаточно средств, и тп)
* accept_transaction(id) code - подвтерждение транзакции. Обновление статуса транзакции на статус accepted (code=0) и добавление записи в balance_changes для destination. В противном случае статус транзакции error и code отражает внештатную ситуацию (транзакция закрыта, source_amount=null и тп)
* cancel_transaction(id) code - отмена транзакции. Обновление статуса транзакции на статус cansel (code=0) и добавление запись в balance_changes для source. В противном случае статус транзакции error и code отражает внештатную ситуация (транзакция закрыта, source_amount=null и тп)
