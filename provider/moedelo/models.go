package moedelo

// TypeKonteragent Тип контрагента Kontragent (=1) - Покупатель/поставщик Buyer (=2) - Покупатель Seller (=3) - Поставщик Other (=4) - Другое = ['Undefined', 'Kontragent', 'Client', 'Partner', 'Other']
type TypeKonteragent int

const (
	Kontragent TypeKonteragent = iota + 1
	Buyer
	Seller
	Other
)

// FormKonteragent Организационная форма контрагента 1 - Юрлицо 2 - ИП 3 - Физлицо 4 - Нерезидент = ['UL', 'IP', 'FL', 'NR']
type FormKonteragent int

const (
	UL FormKonteragent = iota + 1
	IP
	FL
	NR
)

// BillType Тип счета 1 - Обычный, 2 - Счет-договор = ['0', '1', '2']
type BillType int

const (
	Usual BillType = iota + 1
	InvoiceСontract
)

// BillStatus Статус счета 4 - Неоплачен, 5 - Частично оплачен, 6 - Оплачен = ['0', '1', '2', '3', '4', '5', '6']
type BillStatus int

func (b BillStatus) String() string {
	switch b {
	case NotPaid:
		return "not_paid"
	case PartiallyPaid:
		return "partially_paid"
	case Paid:
		return "paid"
	default:
		return ""
	}
}

const (
	NotPaid BillStatus = iota + 4
	PartiallyPaid
	Paid
)

// BillNdsType  Тип начисления НДС 1 - не начислять, 2 - сверху, 3 - в том числе = ['1', '2', '3']
type BillNdsType int

const (
	NotCharge BillStatus = iota + 1
	FromAbove
	Including
)

// SalesType Тип позиции  1 - Товар 2 - Услуга = ['0', '1', '2']
type SalesType int

const (
	Product SalesType = iota + 1
	Service
)

// NdsType  Способ расчёта НДС -1 - Без НДС, 0 - НДС 0%, 0 - НДС 0%, 18 - НДС 18%, 20 - НДС 20% = ['0', '10', '18', '20', '110', '118', '120', '-1']
type NdsType int

const (
	NoNds NdsType = -1
	Nds0  NdsType = 0
	Nds18 NdsType = 18
	Nds20 NdsType = 20
)

// Информация о переводе
type TransferInformation struct {
	Description string
	Email       string
}

type KontragentRepresentationCollection struct {
	ResourceList []KontragentRepresentation `json:"ResourceList"`
}

type KontragentModel struct {
	Inn                 string          `json:"Inn,omitempty"`                 // ИНН
	Ogrn                string          `json:"Ogrn,omitempty"`                // ОГРН
	Okpo                string          `json:"Okpo,omitempty"`                // ОКПО
	Name                string          `json:"Name"`                          // Название или ФИО, если контрагент физ. лицо.
	Type                TypeKonteragent `json:"Type,omitempty"`                // Тип контрагента Kontragent (=1) - Покупатель/поставщик Buyer (=2) - Покупатель Seller (=3) - Поставщик Other (=4) - Другое = ['Undefined', 'Kontragent', 'Client', 'Partner', 'Other']
	Form                FormKonteragent `json:"Form,omitempty"`                // Организационная форма контрагента 1 - Юрлицо 2 - ИП 3 - Физлицо 4 - Нерезидент = ['UL', 'IP', 'FL', 'NR']
	LegalAddress        string          `json:"LegalAddress,omitempty"`        // Юридический адрес
	ActualAddress       string          `json:"ActualAddress,omitempty"`       // Почтовый адрес
	RegistrationAddress string          `json:"RegistrationAddress,omitempty"` //  Адрес регистрации, для физ.лица,
	TaxpayerNumber      string          `json:"TaxpayerNumber,omitempty"`      // Номер налогоплательщика (заполняется только для нерезидентов, вместо инн).Нельзя заполнять одновременно с inn,
	AdditionalRegNumber string          `json:"AdditionalRegNumber,omitempty"` // Дополнительный рег.номер (заполняется только для нерезидентов),
}

type KontragentRepresentation struct {
	ID                  int64           `json:"Id"`                   // Числовой идентификатор контрагента
	Inn                 string          `json:"Inn"`                  // ИНН
	Ogrn                string          `json:"Ogrn"`                 // ОГРН
	Okpo                string          `json:"Okpo"`                 // ОКПО
	Name                string          `json:"Name"`                 // Название или ФИО, если контрагент физ. лицо.
	Type                TypeKonteragent `json:"Type"`                 // Тип контрагента Kontragent (=1) - Покупатель/поставщик Buyer (=2) - Покупатель Seller (=3) - Поставщик Other (=4) - Другое = ['Undefined', 'Kontragent', 'Client', 'Partner', 'Other']
	Form                FormKonteragent `json:"Form"`                 // Организационная форма контрагента 1 - Юрлицо 2 - ИП 3 - Физлицо 4 - Нерезидент = ['UL', 'IP', 'FL', 'NR']
	IsArchived          bool            `json:"IsArchived"`           // Является ли контрагент архивным
	LegalAddress        string          `json:"LegalAddress"`         // Юридический адрес
	ActualAddress       string          `json:"ActualAddress"`        // Почтовый адрес
	SubcontoID          int64           `json:"SubcontoId,omitempty"` // Subconto id
	RegistrationAddress string          `json:"RegistrationAddress" ` //  Адрес регистрации, для физ.лица,
	TaxpayerNumber      string          `json:"TaxpayerNumber"`       // Номер налогоплательщика (заполняется только для нерезидентов, вместо инн).Нельзя заполнять одновременно с inn,
	AdditionalRegNumber string          `json:"AdditionalRegNumber"`  // Дополнительный рег.номер (заполняется только для нерезидентов),
}

type KontragentSettlementAccountRepresentation struct {
	ID                  int64  `json:"Id"`                  // Числовой иденификатор
	Number              string `json:"Number"`              // Номер расчетного счета
	Bik                 string `json:"Bik"`                 // БИК банка (для всех, кроме контрагента нерезидента)
	NonResidentBankName string `json:"NonResidentBankName"` // Назнание банка (только если контрагент нерезидент)
	Comment             string `json:"Comment"`             // Комментарий к расчетному счету
}

type KontragentSettlementAccountModel struct {
	Bik                 string `json:"Bik,omitempty"`                 // БИК банка (для всех, кроме контрагента нерезидента)
	Number              string `json:"Number"`                        // Номер расчетного счета
	NonResidentBankName string `json:"NonResidentBankName,omitempty"` // Назнание банка (только если контрагент нерезидент)
	Comment             string `json:"Comment,omitempty"`             // Комментарий к расчетному счету
}

type KontragentContactRepresentation struct {
	ID     int64                                  `json:"Id"`     // Числовой идентификатор
	Fio    string                                 `json:"Fio"`    // ФИО контакта
	Skype  string                                 `json:"Skype"`  // Skype индетификатор контакта
	Emails []KontragentContactEmailRepresentation `json:"Emails"` // Список Email-ов контакта
	Phones []KontragentContactPhoneRepresentation `json:"Phones"` //  Список телефонов контакта
}
type KontragentContactEmailRepresentation struct {
	ID          int64  `json:"Id"`          // Числовой идентификатор
	Email       string `json:"Email"`       // Email адрес
	Description string `json:"Description"` // Опсиание
}
type KontragentContactPhoneRepresentation struct {
	ID          int64  `json:"Id"`          // Числовой идентификатор
	Number      string `json:"Number"`      // Номер телефона
	Description string `json:"Description"` // Опсиание
}

type KontragentContactModel struct {
	Fio    string                        `json:"Fio"`              // ФИО контакта
	Skype  string                        `json:"Skype,omitempty"`  // Skype индетификатор контакта
	Emails []KontragentContactEmailModel `json:"Emails,omitempty"` // Список Email-ов контакта
	Phones []KontragentContactPhoneModel `json:"Phones,omitempty"` //  Список телефонов контакта
}
type KontragentContactEmailModel struct {
	Email       string `json:"Email,omitempty"`       // Email адрес
	Description string `json:"Description,omitempty"` // Опсиание
}
type KontragentContactPhoneModel struct {
	Number      string `json:"Number,omitempty"`      // Номер телефона
	Description string `json:"Description,omitempty"` // Опсиание
}

type BillSaveRequestModel struct {
	Number              string                   `json:"Number,omitempty"`              // Номер документа Если поле не заполнено, значение будет вычислено автоматически.
	DocDate             string                   `json:"DocDate"`                       // Дата документа
	ProjectId           int64                    `json:"ProjectId,omitempty"`           // Договор с контрагентом
	SettlementAccountId int64                    `json:"SettlementAccountId,omitempty"` // Id расчетного счета
	Type                BillType                 `json:"Type"`                          // Тип счета 1 - Обычный, 2 - Счет-договор = ['0', '1', '2']
	Status              BillStatus               `json:"Status,omitempty"`              // Статус счета 4 - Неоплачен, 5 - Частично оплачен, 6 - Оплачен = ['0', '1', '2', '3', '4', '5', '6']
	KontragentID        int64                    `json:"KontragentId"`                  // Id контрагента
	DeadLine            string                   `json:"DeadLine,omitempty"`            // Дата окончания действия счета
	AdditionalInfo      string                   `json:"AdditionalInfo,omitempty"`      // Дополнительная информация
	ContractSubject     string                   `json:"ContractSubject,omitempty"`     // Предмет договора (для счета-договора)
	NdsPositionType     BillNdsType              `json:"NdsPositionType,omitempty"`     // Тип начисления НДС 1 - не начислять, 2 - сверху, 3 - в том числе = ['1', '2', '3']
	IsCovered           bool                     `json:"IsCovered,omitempty"`           // Статус "Закрыт", полностью закрыт первичными документами
	UseStampAndSign     bool                     `json:"UseStampAndSign,omitempty"`     // Статус "Печать и подпись"
	Items               []SalesDocumentItemModel `json:"Items"`                         // Позиции документа
}
type SalesDocumentItemModel struct {
	Name           string    `json:"Name"`                     // Наименование позиции
	Count          float64   `json:"Count"`                    // Количество
	Unit           string    `json:"Unit"`                     // Единица измерения
	Type           SalesType `json:"Type"`                     // Тип позиции  1 - Товар 2 - Услуга = ['0', '1', '2']
	Price          float64   `json:"Price"`                    // Цена за одну позицию
	NdsType        NdsType   `json:"NdsType,omitempty"`        // Способ расчёта НДС -1 - Без НДС, 0 - НДС 0%, 0 - НДС 0%, 18 - НДС 18%, 20 - НДС 20% = ['0', '10', '18', '20', '110', '118', '120', '-1']
	DiscountRate   float64   `json:"DiscountRate,omitempty"`   // Размер скидки в процентах
	StockProductId int64     `json:"StockProductId,omitempty"` // ID товара или материала (Для услуги поле не передавать)
	SumWithoutNds  float64   `json:"SumWithoutNds,omitempty"`  // Сумма без НДС ,
	NdsSum         float64   `json:"NdsSum,omitempty"`         // Сумма НДС
	SumWithNds     float64   `json:"SumWithNds,omitempty"`     // Cумма с НДС
}

type BillRepresentationCollection struct {
	Count        int64                              `json:"Count,omitempty"`
	ResourceList []BillCollectionItemRepresentation `json:"ResourceList"`
	TotalCount   int64                              `json:"TotalCount,omitempty"`
}

type BillCollectionItemRepresentation struct {
	ID                int64                  `json:"Id"`                          //  Id документа (Сквозная нумерация по всем типам документов)
	Number            string                 `json:"Number"`                      // Номер документа (уникальный в пределах года)
	DocDate           string                 `json:"DocDate"`                     // Дата документа
	Type              BillType               `json:"Type,omitempty"`              // Тип счета 1 - Обычный, 2 - Счет-договор = ['0', '1', '2']
	Status            BillStatus             `json:"Status,omitempty"`            // Статус счета 4 - Неоплачен, 5 - Частично оплачен, 6 - Оплачен = ['0', '1', '2', '3', '4', '5', '6']
	KontragentID      int64                  `json:"KontragentId,omitempty"`      // Id контрагента
	SettlementAccount SettlementAccountModel `json:"SettlementAccount,omitempty"` // Расчетный счет
	ProjectID         int64                  `json:"ProjectId,omitempty"`         // Договор с контрагентом
	DeadLine          string                 `json:"DeadLine,omitempty"`          // Дата окончания действия счета
	AdditionalInfo    string                 `json:"AdditionalInfo,omitempty"`    // Дополнительная информация
	ContractSubject   string                 `json:"ContractSubject,omitempty"`   // Предмет договора (для счета-договора)
	NdsPositionType   int64                  `json:"NdsPositionType,omitempty"`   // Тип начисления НДС 1 - не начислять, 2 - сверху, 3 - в том числе = ['1', '2', '3']
	IsCovered         bool                   `json:"IsCovered,omitempty"`         // Статус "Закрыт", полностью закрыт первичными документами
	Sum               float64                `json:"Sum,omitempty"`               // Сумма документа
	PaidSum           float64                `json:"PaidSum,omitempty"`           // Поступившая оплата
}

type BillRepresentation struct {
	ID                int64                             `json:"Id"`                          // Id документа (Сквозная нумерация по всем типам документов)
	Number            string                            `json:"Number"`                      // Номер документа (уникальный в пределах года)
	DocDate           string                            `json:"DocDate"`                     // Дата документа
	Items             []SalesDocumentItemRepresentation `json:"Items,omitempty"`             // Позиции документа
	Online            string                            `json:"Online,omitempty"`            // Ссылка на счет для партнеров
	Context           Context                           `json:"Context,omitempty"`           // Информация об изменениях документа
	Payments          []BillPayment                     `json:"Payments,omitempty"`          // Платежи, связанные со счётом
	Type              BillType                          `json:"Type,omitempty"`              // Тип счета 1 - Обычный, 2 - Счет-договор = ['0', '1', '2']
	Status            BillStatus                        `json:"Status,omitempty"`            // Статус счета 4 - Неоплачен, 5 - Частично оплачен, 6 - Оплачен = ['0', '1', '2', '3', '4', '5', '6']
	KontragentID      int64                             `json:"KontragentId,omitempty"`      // Id контрагента
	SettlementAccount SettlementAccountModel            `json:"SettlementAccount,omitempty"` // Расчетный счет
	ProjectID         int64                             `json:"ProjectId,omitempty"`         // Договор с контрагентом
	DeadLine          string                            `json:"DeadLine,omitempty"`          // Дата окончания действия счета
	AdditionalInfo    string                            `json:"AdditionalInfo,omitempty"`    // Дополнительная информация
	ContractSubject   string                            `json:"ContractSubject,omitempty"`   // Предмет договора (для счета-договора)
	NdsPositionType   BillNdsType                       `json:"NdsPositionType,omitempty"`   // Тип начисления НДС 1 - не начислять, 2 - сверху, 3 - в том числе = ['1', '2', '3']
	IsCovered         bool                              `json:"IsCovered,omitempty"`         // Статус "Закрыт", полностью закрыт первичными документами
	Sum               float64                           `json:"Sum,omitempty"`               // Сумма документа
	PaidSum           float64                           `json:"PaidSum,omitempty"`           // Поступившая оплата
}

type SalesDocumentItemRepresentation struct {
	DiscountRate   float64   `json:"DiscountRate"`            // Размер скидки в процентах
	ID             int64     `json:"Id"`                      // Идентификатор
	Name           string    `json:"Name"`                    // Наименование позиции
	Count          float64   `json:"Count"`                   // Количество
	Unit           string    `json:"Unit"`                    // Единица измерения
	Type           SalesType `json:"Type"`                    // Тип позиции 1 - Товар 2 - Услуга = ['0', '1', '2']
	Price          float64   `json:"Price"`                   // Цена за одну позицию
	NdsType        NdsType   `json:"NdsType"`                 // Способ расчёта НДС -1 - Без НДС, 0 - НДС 0%, 0 - НДС 0%, 18 - НДС 18% = ['0', '10', '18', '20', '110', '118', '120', '-1']
	SumWithoutNds  float64   `json:"SumWithoutNds,omitempty"` // Сумма без НДС; если указана, то используется в расчёте Суммы НДС и Суммы с НДС, если они пустые
	NdsSum         float64   `json:"NdsSum,omitempty"`        // Сумма НДС; если указана, то используется в расчёте Суммы без НДС и Суммы с НДС, если они пустые
	SumWithNds     float64   `json:"SumWithNds,omitempty"`    // Cумма с НДС; если указана, то используется в расчёте Суммы НДС и Суммы без НДС, если они пустые
	StockProductId int64     `json:"StockProductId"`          // Товар/материал
}

type Context struct {
	CreateDate string `json:"CreateDate"` // Дата создания
	ModifyDate string `json:"ModifyDate"` // Дата последнего изменения
	ModifyUser string `json:"ModifyUser"` // Пользователь, вносивший изменения в объект последним
}

type BillPayment struct {
	Number string  `json:"Number,omitempty"` // Номер платёжного документа
	Date   string  `json:"Date,omitempty"`   // Дата платежа
	Sum    float64 `json:"Sum,omitempty"`    // Сумма
	ID     float64 `json:"Id,omitempty"`     // Идентификатор платежа
}

type SettlementAccountModel struct {
	AccountID     int64  `json:"AccountId,omitempty"`     // Id счета
	AccountNumber string `json:"AccountNumber,omitempty"` //  Номер счета
}
