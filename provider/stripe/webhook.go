package stripe

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo"
	"github.com/stripe/stripe-go/webhook"
	"go.uber.org/zap"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
)

// If you are testing your webhook locally with the Stripe CLI you
// can find the endpoint's secret by running `stripe trigger`
// Otherwise, find your endpoint's secret in your webhook settings in the Developer Dashboard
var endpointSecret = "whsec_IDoODms1QL7LRHhEpmwRuwAmp3lejhKi"

// StripeWebhookHandler обработчик вебхука от stripe.
func (p *Provider) StripeWebhookHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		const MaxBodyBytes = int64(65536)
		c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, MaxBodyBytes)
		payload, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading request body: %v\n", err)
			c.Response().WriteHeader(http.StatusServiceUnavailable)
			return nil
		}

		// Pass the request body and Stripe-Signature header to ConstructEvent, along
		// with the webhook signing key.
		event, err := webhook.ConstructEvent(payload, c.Request().Header.Get("Stripe-Signature"),
			endpointSecret)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying webhook signature: %v\n", err)
			c.Response().WriteHeader(http.StatusBadRequest) // Return a 400 error on a bad signature
			return nil
		}

		// Unmarshal the event data into an appropriate struct depending on its Type
		switch event.Type {
		case "payment_intent.succeeded":
			var tr engine.Transaction
			err := p.db.SelectOneTo(&tr, "WHERE provider = $1 AND provider_oper_id = $2", STRIPE, event.GetObjectValue("id"))
			if err != nil {
				p.l.Warn(
					"Failed get transaction by external order from sberbank payment",
					zap.String("extOrderID", event.GetObjectValue("id")),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return err
			}
			str := strategies.GetTransactionStrategy(strategies.ExistTrName(tr.Strategy))
			if str == nil {
				p.l.Error("Transaction strategy not found.", zap.Int64("tx_id", tr.TransactionID))
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			switch tr.Status {
			case engine.WACCEPTED_TX, engine.ACCEPTED_TX:
				// Было списание захолдированных средсть, и преходв в ACCEPTED_TX, обработка не требуется
				return nil
			case engine.AUTH_TX:
			default:
				p.l.Error("invoice is not ready for pay", zap.Int64("tx_id", tr.TransactionID))
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			status := event.GetObjectValue("status")
			tr.ProviderOperStatus = &status
			if err := p.db.Save(&tr); err != nil {
				p.l.Error(
					"Failed save transaction from status stripe payment",
					zap.String("extOrderID", event.GetObjectValue("id")),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			// Отправляем сообщение на переход транзакции в ACCEPTED
			if err := p.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, strategies.MessageUpdateTransaction{
				ClientID:      tr.ClientID,
				TransactionID: tr.TransactionID,
				Strategy:      tr.Strategy,
				Status:        engine.ACCEPTED_TX,
			}); err != nil {
				p.l.Error(
					"failed send accept transaction",
					zap.Int64("tx_id", tr.TransactionID),
					zap.String("extOrderID", event.GetObjectValue("id")),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}

		case "payment_intent.amount_capturable_updated":
			var tr engine.Transaction
			err := p.db.SelectOneTo(&tr, "WHERE provider = $1 AND provider_oper_id = $2", STRIPE, event.GetObjectValue("id"))
			if err != nil {
				p.l.Warn(
					"Failed get transaction by external order from sberbank payment",
					zap.String("extOrderID", event.GetObjectValue("id")),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return err
			}
			str := strategies.GetTransactionStrategy(strategies.ExistTrName(tr.Strategy))
			if str == nil {
				p.l.Error("Transaction strategy not found.", zap.Int64("tx_id", tr.TransactionID))
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			if tr.Status != engine.AUTH_TX {
				p.l.Error(
					"invoice is not ready for pay",
					zap.Int64("tx_id", tr.TransactionID),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			status := event.GetObjectValue("status")
			tr.ProviderOperStatus = &status
			if err := p.db.Save(&tr); err != nil {
				p.l.Error(
					"Failed save transaction from status sberbank payment",
					zap.Int64("tx_id", tr.TransactionID),
					zap.String("extOrderID", event.GetObjectValue("id")),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			if err := p.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, strategies.MessageUpdateTransaction{
				ClientID:      tr.ClientID,
				TransactionID: tr.TransactionID,
				Strategy:      tr.Strategy,
				Status:        engine.HOLD_TX,
			}); err != nil {
				p.l.Error(
					"Failed publish received funds by order",
					zap.Int64("tx_id", tr.TransactionID),
					zap.String("extOrderID", event.GetObjectValue("id")),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
		case "payment_intent.payment_failed":
			var tr engine.Transaction
			err := p.db.SelectOneTo(&tr, "WHERE provider = $1 AND provider_oper_id = $2", STRIPE, event.GetObjectValue("id"))
			if err != nil {
				p.l.Warn(
					"Failed get transaction by external order from sberbank payment",
					zap.String("extOrderID", event.GetObjectValue("id")),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return err
			}
			str := strategies.GetTransactionStrategy(strategies.ExistTrName(tr.Strategy))
			if str == nil {
				p.l.Error("Transaction strategy not found.", zap.Int64("tx_id", tr.TransactionID))
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}

			status := event.GetObjectValue("status")
			tr.ProviderOperStatus = &status
			if err := p.db.Save(&tr); err != nil {
				p.l.Error(
					"Failed save transaction from status sberbank payment",
					zap.String("provider_intent_id", event.GetObjectValue("id")),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
			if err := p.nc.Publish(strategies.UPDATE_TRANSACTION_SUBJECT, strategies.MessageUpdateTransaction{
				ClientID:      tr.ClientID,
				TransactionID: tr.TransactionID,
				Strategy:      tr.Strategy,
				Status:        engine.REJECTED_TX,
			}); err != nil {
				p.l.Error(
					"failed accept invoice",
					zap.String("provider_intent_id", event.GetObjectValue("id")),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().WriteHeader(http.StatusOK)
				return nil
			}
		default:
			p.l.Error(
				"Unexpected event",
				zap.String("event_id", event.ID),
				zap.String("event_type", event.Type),
				zap.String("raw", string(event.Data.Raw)),
				zap.Error(err),
			)
			c.Response().WriteHeader(http.StatusOK)
			return nil
		}

		c.Response().WriteHeader(http.StatusOK)
		return nil
	}
}
