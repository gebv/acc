package sberbank

import (
	"context"
	"encoding/json"
	"net/http"

	"cloud.google.com/go/pubsub"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
)

// WebhookHandler обработчик вебхука от сбербанк.
func (p *Provider) WebhookHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		callback := c.QueryParam("callback")
		if p == nil {
			c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
			c.Response().WriteHeader(http.StatusTemporaryRedirect)
			return errors.New("CardSberbank system is not configured.")
		}
		result := c.QueryParam("result")
		switch result {
		case "success":
			return p.successSberbankWebhookHandler(c)
		case "fail":
			return p.failSberbankWebhookHandler(c)
		}
		c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
		c.Response().WriteHeader(http.StatusTemporaryRedirect)
		return nil
	}
}

// successSberbankWebhookHandler запрос в сбербанк для проверки статуса заказа, пополнения счета и оплаты заказа
func (p *Provider) successSberbankWebhookHandler(c echo.Context) error {
	//securityCode := c.QueryParam("security_code")
	extOrderID := c.QueryParam("orderId")
	callback := c.QueryParam("callback")

	var tr engine.Transaction
	err := p.db.SelectOneTo(&tr, "WHERE provider = $1 AND provider_oper_id = $2", SBERBANK, extOrderID)
	if err != nil {
		p.l.Warn(
			"Failed get transaction by external order from sberbank payment",
			zap.String("extOrderID", extOrderID),
			zap.Error(err),
		)
		c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
		c.Response().WriteHeader(http.StatusTemporaryRedirect)
		return err
	}
	// TODO добавить в транзакцию Payload где будет код хранится для запроса,
	//  после обработки убираем и повторный запрос уже не пройдет
	//if it.SecurityCode != securityCode {
	//	p.l.Warn(
	//		"Failed get invoice by external code from sberbank payment",
	//		zap.String("extOrderID", extOrderID),
	//		zap.Error(err),
	//	)
	//	c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
	//	c.Response().WriteHeader(http.StatusTemporaryRedirect)
	//	return errors.New("Failed security code")
	//}

	str := strategies.GetTransactionStrategy(strategies.ExistTrName(tr.Strategy))
	if str == nil {
		c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
		c.Response().WriteHeader(http.StatusTemporaryRedirect)
		return errors.New("Transaction strategy not found.")
	}

	paymentInfo, err := p.GetOrderStatus(extOrderID)
	if err != nil {
		p.l.Warn("Failed get order status from sberbank",
			zap.Error(err),
		)
		c.Response().Header().Set("Location", callback+"?payment_state=fail")
		c.Response().WriteHeader(http.StatusTemporaryRedirect)
		return err
	}
	if paymentInfo.UpdateStatus {
		switch paymentInfo.PaymentAmountInfo.PaymentState {
		case DEPOSITED: // Статус подтверждает списание средств
			if tr.Status != engine.AUTH_TX {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.New("invoice is not ready for pay")
			}
			status := DEPOSITED
			tr.ProviderOperStatus = &status
			if err := p.db.Save(&tr); err != nil {
				p.l.Warn(
					"Failed save transaction from status sberbank payment",
					zap.String("extOrderID", extOrderID),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return err
			}
			// Отправляем сообщение на переход транзакции в ACCEPTED
			b, err := json.Marshal(&strategies.MessageUpdateTransaction{
				ClientID:      tr.ClientID,
				TransactionID: tr.TransactionID,
				Strategy:      tr.Strategy,
				Status:        engine.ACCEPTED_TX,
			})
			if err != nil {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.Wrap(err, "failed json marshal for publish accept transaction")
			}
			if _, err := p.pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(context.Background(), &pubsub.Message{
				Data: b,
			}).Get(context.Background()); err != nil {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.Wrap(err, "failed publish accept transaction")
			}
		case APPROVED: // Статус подтверждает холдирования средств
			if tr.Status != engine.AUTH_TX {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.New("invoice is not ready for pay")
			}
			status := APPROVED
			tr.ProviderOperStatus = &status
			if err := p.db.Save(&tr); err != nil {
				p.l.Warn(
					"Failed save transaction from status sberbank payment",
					zap.String("extOrderID", extOrderID),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return err
			}
			b, err := json.Marshal(&strategies.MessageUpdateTransaction{
				ClientID:      tr.ClientID,
				TransactionID: tr.TransactionID,
				Strategy:      tr.Strategy,
				Status:        engine.HOLD_TX,
			})
			if err != nil {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.Wrap(err, "Failed json marshal for publish received funds by order")
			}
			if _, err := p.pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(context.Background(), &pubsub.Message{
				Data: b,
			}).Get(context.Background()); err != nil {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.Wrap(err, "Failed publish received funds by order")
			}
		default:
			p.l.Warn(
				"SberbankWebHookSuccess: not processed status",
				zap.String("order_number", paymentInfo.OrderNumber),
				zap.String("status", paymentInfo.PaymentAmountInfo.PaymentState),
			)
			c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
			c.Response().WriteHeader(http.StatusTemporaryRedirect)
			return errors.New("Failed payment status.")
		}
	}
	c.Response().Header().Set("Location", callback+"?payment_state=success")
	c.Response().WriteHeader(http.StatusTemporaryRedirect)
	return nil
}

// SberbankWebHookFail не прошла оплате
func (p *Provider) failSberbankWebhookHandler(c echo.Context) error {
	//securityCode := c.QueryParam("security_code")
	extOrderID := c.QueryParam("orderId")
	callback := c.QueryParam("callback")

	var tr engine.Transaction
	err := p.db.SelectOneTo(&tr, "WHERE provider = $1 AND provider_oper_id = $2", SBERBANK, extOrderID)
	if err != nil {
		p.l.Warn(
			"Failed get invoice transaction by external code from sberbank payment",
			zap.String("extOrderID", extOrderID),
			zap.Error(err),
		)
		c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
		c.Response().WriteHeader(http.StatusTemporaryRedirect)
		return err
	}
	c.Response().Header().Set("Location", callback+"?payment_state=fail")
	c.Response().WriteHeader(http.StatusTemporaryRedirect)
	paymentInfo, err := p.GetOrderStatus(extOrderID)
	if err != nil {
		p.l.Warn("Failed get order status from sberbank",
			zap.Error(err),
		)
		c.Response().Header().Set("Location", callback+"?payment_state=fail")
		c.Response().WriteHeader(http.StatusTemporaryRedirect)
		return err
	}
	if paymentInfo.UpdateStatus {
		switch paymentInfo.PaymentAmountInfo.PaymentState {
		case DECLINED:
			status := DECLINED
			tr.ProviderOperStatus = &status
			if err := p.db.Save(&tr); err != nil {
				p.l.Warn(
					"Failed save transaction from status sberbank payment",
					zap.String("extOrderID", extOrderID),
					zap.String("status", status),
					zap.Error(err),
				)
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return err
			}
			b, err := json.Marshal(&strategies.MessageUpdateTransaction{
				ClientID:      tr.ClientID,
				TransactionID: tr.TransactionID,
				Strategy:      tr.Strategy,
				Status:        engine.REJECTED_TX,
			})
			if err != nil {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.Wrap(err, "Failed json marshal for accept invoice")
			}
			if _, err := p.pb.Topic(strategies.UPDATE_TRANSACTION_SUBJECT).Publish(context.Background(), &pubsub.Message{
				Data: b,
			}).Get(context.Background()); err != nil {
				c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
				c.Response().WriteHeader(http.StatusTemporaryRedirect)
				return errors.Wrap(err, "Failed accept invoice")
			}
		default:
			p.l.Warn(
				"SberbankWebHookSuccess: not processed status",
				zap.String("order_number", paymentInfo.OrderNumber),
				zap.String("status", paymentInfo.PaymentAmountInfo.PaymentState),
			)
			c.Response().Header().Set("Location", callback+"?payment_state=internal_fail")
			c.Response().WriteHeader(http.StatusTemporaryRedirect)
			return errors.New("Failed payment status.")
		}
	}
	p.l.Debug(
		"SberbankWebHookFail",
		zap.String("extOrderID", extOrderID),
		zap.String("order_number", paymentInfo.OrderNumber),
		zap.Any("payment_info", paymentInfo),
	)
	return nil
}
