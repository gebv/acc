package stripe

import (
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/paymentintent"
	"github.com/stripe/stripe-go/paymentmethod"
	"github.com/stripe/stripe-go/setupintent"

	"github.com/gebv/acca/provider"
)

func NewProvider(db *reform.DB, nc *nats.EncodedConn) *Provider {
	return &Provider{
		db: db,
		nc: nc,
		s: &provider.Store{
			DB: db,
		},
		l: zap.L().Named("stripe_provider"),
	}
}

type Provider struct {
	db *reform.DB
	nc *nats.EncodedConn
	s  *provider.Store
	l  *zap.Logger
}

const (
	STRIPE provider.Provider = "stripe"
)

// Создает клиента
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

// Сохранение карты без платежа.
func (p *Provider) SetupIntent() (string, string, error) {
	intent, err := setupintent.New(&stripe.SetupIntentParams{}) // TODO проверить с nil
	if err != nil {
		p.l.Warn(
			"Failed setup intent",
			zap.Error(err),
		)
		return "", "", errors.Wrap(err, "Failed setup intent")
	}
	return intent.ID, intent.ClientSecret, nil
}

// Добавляем метод платежа (карты) к пользователю
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
	customerID *string,
	pmID *string,
	confirm *bool,
) (*stripe.PaymentIntent, error) {
	paymentIntent, err := paymentintent.New(&stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(string(currency)),
		PaymentMethodTypes: []*string{
			stripe.String("card"),
		},
		Customer:      customerID,
		PaymentMethod: pmID,
		Confirm:       confirm, // Если карта обязательно подтверждение 3D secure, то будет ошибка.
		// OffSession:    stripe.Bool(true), // Баз Confirm не указывать, ошибка.
		//SetupFutureUsage: stripe.String(string(stripe.PaymentIntentSetupFutureUsageOffSession)), // С последующим использованием карты.
	})
	if err != nil {
		p.l.Warn(
			"Failed payment intent",
			zap.String("customer_id", stripe.StringValue(customerID)),
			zap.String("payment_method", stripe.StringValue(pmID)),
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

func (p *Provider) Confirm(paymentIntentID string, pmID *string) error {
	var paymentIntent *stripe.PaymentIntentConfirmParams
	if pmID != nil {
		paymentIntent = &stripe.PaymentIntentConfirmParams{
			PaymentMethod: pmID,
			// OffSession:    stripe.Bool(true), // Если для карты обязательно 3D secure будет ошибка
		}
	}
	_, err := paymentintent.Confirm(
		paymentIntentID,
		paymentIntent,
	)
	if err != nil {
		p.l.Warn(
			"Failed confirm payment intent",
			zap.String("payment_intent_id", paymentIntentID),
			zap.String("payment_method", stripe.StringValue(pmID)),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed confirm payment intent")
	}
	return nil
}

// TODO добавить метот отмены холда
func (p *Provider) ReverseForHold() error {
	return nil
}

// TODO добавить метот отмены холда
func (p *Provider) Refund() error {
	return nil
}
