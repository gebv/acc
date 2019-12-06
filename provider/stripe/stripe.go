package stripe

import (
	"os"

	"cloud.google.com/go/firestore"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/paymentintent"
	"github.com/stripe/stripe-go/paymentmethod"
	"github.com/stripe/stripe-go/refund"
	"github.com/stripe/stripe-go/setupintent"

	"github.com/gebv/acca/provider"
)

func NewProvider(db *reform.DB, fs *firestore.Client) *Provider {
	stripe.Key = os.Getenv("STRIPE_KEY")
	endpointSecret = os.Getenv("STRIPE_ENDPOINT_SECRET")
	if os.Getenv("STRIPE_KEY") == "" || endpointSecret == "" {
		panic("Stripe env is empty.")
	}
	return &Provider{
		db: db,
		fs: fs,
		s: &provider.Store{
			DB: db,
		},
		l: zap.L().Named("stripe_provider"),
	}
}

type Provider struct {
	fs *firestore.Client
	db *reform.DB
	s  *provider.Store
	l  *zap.Logger
}

const (
	STRIPE provider.Provider = "stripe"
)

// Создает клиента в stripe и возвращает customerID
func (p *Provider) CreateCustomer(email, name, pmID *string) (string, error) {
	cs, err := customer.New(&stripe.CustomerParams{
		Email:         email,
		Name:          name,
		PaymentMethod: pmID,
	})
	if err != nil {
		p.l.Warn(
			"Failed create customer",
			zap.String("email", stripe.StringValue(email)),
			zap.String("name", stripe.StringValue(name)),
			zap.String("payment_method", stripe.StringValue(pmID)),
			zap.Error(err),
		)
		return "", errors.Wrap(err, "Failed create customer")
	}
	return cs.ID, nil
}

// Сохранение карты без платежа, создает клиентский секрет,
//при использовании которого можно добавить к пользователю карту на стороне клиента.
// Если не указаны все параметры, то нужно на стороне клиента использовать клиентский секрет (2 параметр)
// и подтвердить данные карты, а далее использовав метода AttachPaymentMethodToCustomer добавить полученный
// платежный метод. Если указать все параметры и выставить флаг confirm в true, то платежный метод
// будет сразу добавлен пользователю.
func (p *Provider) SetupIntent(customerID, pmID string, confirm bool) (string, string, error) {
	var ptrCustomerID *string
	if customerID != "" {
		ptrCustomerID = &customerID
	}
	var ptrPmID *string
	if pmID != "" {
		ptrPmID = &pmID
	}
	intent, err := setupintent.New(&stripe.SetupIntentParams{
		Customer:      ptrCustomerID,
		PaymentMethod: ptrPmID,
		Confirm:       stripe.Bool(confirm),
	})
	if err != nil {
		p.l.Warn(
			"Failed setup intent",
			zap.Error(err),
		)
		return "", "", errors.Wrap(err, "Failed setup intent")
	}
	return intent.ID, intent.ClientSecret, nil
}

// Добавляем платежного метода (карты) к пользователю. id берести при подтверждении клиентом выставленного счетоа
// по клиентскому секрету.
func (p *Provider) AttachPaymentMethodToCustomer(pmID, customerID string) error {
	_, err := paymentmethod.Attach(pmID, &stripe.PaymentMethodAttachParams{
		Customer: &customerID,
	})
	if err != nil {
		p.l.Warn(
			"Failed create customer",
			zap.String("customer_id", customerID),
			zap.String("payment_method", pmID),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed create customer")
	}
	return nil
}

// Выставляем счет клиенту.
func (p *Provider) PaymentIntent(
	amount int64,
	currency stripe.Currency,
	customerID string,
	pmID string,
	confirm bool,
) (*stripe.PaymentIntent, error) {
	var ptrCustomerID *string
	if customerID != "" {
		ptrCustomerID = &customerID
	}
	var ptrPmID *string
	if pmID != "" {
		ptrPmID = &pmID
	}
	paymentIntent, err := paymentintent.New(&stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(string(currency)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		Customer:      ptrCustomerID,
		PaymentMethod: ptrPmID,
		Confirm:       stripe.Bool(confirm), // Если карта обязательно подтверждение 3D secure, то будет ошибка.
		// OffSession:    stripe.Bool(true), // Баз Confirm не указывать, ошибка.
		//SetupFutureUsage: stripe.String(string(stripe.PaymentIntentSetupFutureUsageOffSession)), // С последующим использованием карты. может быть указан на клиенте
	})
	if err != nil {
		p.l.Warn(
			"Failed payment intent",
			zap.String("customer_id", customerID),
			zap.String("payment_method", pmID),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed payment intent")
	}
	err = p.s.NewOrder(paymentIntent.ID, STRIPE, string(paymentIntent.Status))
	if err != nil {
		return nil, errors.Wrap(err, "Failed insert stripe payment intent")
	}
	return paymentIntent, nil
}

// Выставляем счет клиенту с холдированием.
func (p *Provider) PaymentIntentWithHold(
	amount int64,
	currency stripe.Currency,
	customerID string,
	pmID string,
	confirm bool,
) (*stripe.PaymentIntent, error) {
	var ptrCustomerID *string
	if customerID != "" {
		ptrCustomerID = &customerID
	}
	var ptrPmID *string
	if pmID != "" {
		ptrPmID = &pmID
	}
	pi, err := paymentintent.New(&stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(string(currency)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		Customer:      ptrCustomerID,
		PaymentMethod: ptrPmID,
		CaptureMethod: stripe.String("manual"), // TODO проверить, если будет указано Confirm может нужно убрать manual
		Confirm:       stripe.Bool(confirm),    // Если карта обязательно подтверждение 3D secure, то будет ошибка.
		// OffSession:    stripe.Bool(true), // Баз Confirm не указывать, ошибка.
		//SetupFutureUsage: stripe.String(string(stripe.PaymentIntentSetupFutureUsageOffSession)), // С последующим использованием карты. может быть указана на клиенту.
	})
	if err != nil {
		p.l.Warn(
			"Failed payment intent",
			zap.String("customer_id", customerID),
			zap.String("payment_method", pmID),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed payment intent")
	}
	err = p.s.NewOrder(pi.ID, STRIPE, string(pi.Status))
	if err != nil {
		return nil, errors.Wrap(err, "Failed insert stripe payment intent")
	}
	return pi, nil
}

func (p *Provider) Confirm(paymentIntentID string, pmID string) error {
	var paymentIntent *stripe.PaymentIntentConfirmParams
	if pmID != "" {
		paymentIntent = &stripe.PaymentIntentConfirmParams{
			PaymentMethod: stripe.String(pmID),
			// OffSession:    stripe.Bool(true), // Если для карты обязательно 3D secure будет ошибка
		}
	}
	pi, err := paymentintent.Confirm(
		paymentIntentID,
		paymentIntent,
	)
	if err != nil {
		p.l.Warn(
			"Failed confirm payment intent",
			zap.String("payment_intent_id", paymentIntentID),
			zap.String("payment_method", pmID),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed confirm payment intent")
	}
	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		err = errors.New("failed_status_payment_intent")
		p.l.Warn(
			"Failed confirm status payment intent",
			zap.String("payment_intent_id", paymentIntentID),
			zap.String("status", string(pi.Status)),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// подтвердить захолдированные средства
func (p *Provider) Capture(paymentIntentID string, amount int64) error {
	pi, err := paymentintent.Capture(paymentIntentID, &stripe.PaymentIntentCaptureParams{
		AmountToCapture: stripe.Int64(amount),
	})
	if err != nil {
		p.l.Warn(
			"Failed capture payment intent",
			zap.String("payment_intent_id", paymentIntentID),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed capture payment intent")
	}
	if pi.Status != stripe.PaymentIntentStatusSucceeded {
		err = errors.New("failed_status_payment_intent")
		p.l.Warn(
			"Failed capture status payment intent",
			zap.String("payment_intent_id", paymentIntentID),
			zap.String("status", string(pi.Status)),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (p *Provider) Cancel(paymentIntentID string) error {
	pi, err := paymentintent.Cancel(
		paymentIntentID,
		nil,
	)
	if err != nil {
		p.l.Warn(
			"Failed cancel payment intent",
			zap.String("payment_intent_id", paymentIntentID),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed cancel payment intent")
	}
	if pi.Status != stripe.PaymentIntentStatusCanceled {
		err = errors.New("failed_status_payment_intent")
		p.l.Warn(
			"Failed cancel status payment intent",
			zap.String("payment_intent_id", paymentIntentID),
			zap.String("status", string(pi.Status)),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// Возврат средств от charge
func (p *Provider) Refund(chargeID string, amount *int64) error {
	re, err := refund.New(&stripe.RefundParams{
		Charge: stripe.String(chargeID),
		Amount: amount,
	})
	if err != nil {
		p.l.Warn(
			"Failed refund charge",
			zap.String("charge_id", chargeID),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed refund charge")
	}
	if re.Status != stripe.RefundStatusSucceeded {
		err = errors.New("failed_status_refund_charge")
		p.l.Warn(
			"Failed cancel status payment intent",
			zap.String("charge_id", chargeID),
			zap.String("status", string(re.Status)),
			zap.Error(err),
		)
		return err
	}
	return nil
}

// Get возвращает структуру выставленной платежки(счета).
func (p *Provider) GetPaymentIntent(
	piID string,
) (*stripe.PaymentIntent, error) {
	paymentIntent, err := paymentintent.Get(piID, nil)
	if err != nil {
		p.l.Warn(
			"Failed get payment intent",
			zap.String("payment_intent_id", piID),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed payment intent")
	}
	return paymentIntent, nil
}
