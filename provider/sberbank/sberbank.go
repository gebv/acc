package sberbank

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"

	"github.com/gebv/acca/provider"
)

func NewProvider(db *reform.DB, cfg Config, pb *pubsub.Client) *Provider {
	return &Provider{
		cfg: cfg,
		db:  db,
		pb:  pb,
		s: &provider.Store{
			DB: db,
		},
		l: zap.L().Named("sberbank_provider"),
	}
}

type Provider struct {
	cfg Config
	db  *reform.DB
	pb  *pubsub.Client
	s   *provider.Store
	l   *zap.Logger
}

const (
	SBERBANK provider.Provider = "sberbank"

	CREATED   = "CREATED"
	APPROVED  = "APPROVED"
	DEPOSITED = "DEPOSITED"
	DECLINED  = "DECLINED"
	REVERSED  = "REVERSED"
	REFUNDED  = "REFUNDED"
)

type SberbankOrderStatus struct {
	OrderNumber       string            `json:"orderNumber"`
	PaymentAmountInfo PaymentAmountInfo `json:"paymentAmountInfo"`
	UpdateStatus      bool              `json:"update_status"`
}

type PaymentAmountInfo struct {
	PaymentState    string `json:"paymentState"`
	ApprovedAmount  int64  `json:"approvedAmount"`
	DepositedAmount int64  `json:"depositedAmount"`
}

type SberbankRegister struct {
	OrderID      string `json:"orderId"`
	FormURL      string `json:"formUrl"`
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

type SberbankResp struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// AuthTransfer выполнение авторизации платежа на стороне Сбербанк.
// Возвращает первым значением orderID (внутренний ID в системе сбербанка),
// вторым значением url для выполнения подтверждения от пользователя на стороне сбербанк.
func (p *Provider) AuthTransfer(
	amount int64,
	info TransferInformation,
	hold bool,
) (string, string, error) {
	orderNumber := newPaymentOrderID()

	method := "register.do"
	if hold {
		method = "registerPreAuth.do"
	}
	_url, _ := url.Parse(p.cfg.EntrypointURL + "/payment/rest/" + method)
	q := _url.Query()
	q.Add("amount", strconv.FormatInt(amount, 10))
	q.Add("currency", "643")
	q.Add("language", "ru")
	q.Add("orderNumber", orderNumber)
	q.Add("returnUrl", info.ReturnURL)
	q.Add("failUrl", info.FailURL)
	q.Add("jsonParams", fmt.Sprintf(`{"email":"%s"}`, info.Email))
	q.Add("token", p.cfg.Token)
	_url.RawQuery = q.Encode()

	var rco SberbankRegister
	res, err := http.Get(_url.String())
	if err != nil {
		p.l.Warn(
			"register: get url",
			zap.String("url", _url.String()),
			zap.Error(err),
		)
		return "", "", errors.Wrap(err, "Failed http get sberbank url")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		p.l.Warn(
			"register: read body",
			zap.Error(err),
		)
		return "", "", errors.Wrap(err, "Failed read body response from sberbank")
	}

	err = json.Unmarshal(body, &rco)
	if err != nil {
		p.l.Warn(
			"register: bad unmarshal request from sberbank",
			zap.String("body", string(body)),
			zap.Error(err),
		)
		return "", "", errors.Wrap(err, "Failed unmarshal response from sberbank")
	}
	switch rco.ErrorCode {
	case "":
	default:
		return "", "", errors.New(rco.ErrorMessage)
	}
	err = p.s.NewOrder(rco.OrderID, SBERBANK, CREATED)
	if err != nil {
		return "", "", errors.Wrap(err, "Failed insert sberbank order")
	}
	return rco.OrderID, rco.FormURL, nil
}

func (p *Provider) GetOrderStatus(orderID string) (*SberbankOrderStatus, error) {
	_url, _ := url.Parse(p.cfg.EntrypointURL + "/payment/rest/getOrderStatusExtended.do")
	q := _url.Query()
	q.Add("orderId", orderID)
	q.Add("token", p.cfg.Token)
	_url.RawQuery = q.Encode()

	var os SberbankOrderStatus
	res, err := http.Get(_url.String())
	if err != nil {
		p.l.Warn(
			"orderStatus: get url",
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http get sberbank url")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		p.l.Warn(
			"requestForOrderStatus: read body",
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed read body response from sberbank")
	}
	err = json.Unmarshal(body, &os)
	if err != nil {
		p.l.Warn(
			"register: bad unmarshal request from sberbank",
			zap.String("body", string(body)),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed unmarshal response from sberbank")
	}
	so, err := p.s.GetByOrderID(orderID, SBERBANK)
	if err != nil {
		p.l.Warn(
			"requestForOrderStatus: reload extOrder status",
			zap.String("ext_order_id", orderID),
			zap.Error(err),
		)
		return nil, err
	}
	if so.RawOrderStatus != os.PaymentAmountInfo.PaymentState {
		os.UpdateStatus = true
		err = p.s.SetStatus(orderID, SBERBANK, os.PaymentAmountInfo.PaymentState)
		if err != nil {
			p.l.Warn(
				"requestForOrderStatus: save extOrder status",
				zap.String("ext_order_id", orderID),
				zap.String("status", os.PaymentAmountInfo.PaymentState),
				zap.Error(err),
			)
			return nil, err
		}
	}
	return &os, nil
}

func (p *Provider) DepositForHold(orderID string, amount int64) error {
	_url, _ := url.Parse(p.cfg.EntrypointURL + "/payment/rest/deposit.do")
	q := _url.Query()
	q.Add("orderId", orderID)
	q.Add("amount", strconv.FormatInt(amount, 10))
	q.Add("password", p.cfg.Password)
	q.Add("userName", p.cfg.UserName)
	_url.RawQuery = q.Encode()

	var sr SberbankResp
	res, err := http.Get(_url.String())
	if err != nil {
		zap.L().Warn(
			"deposit: get url",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed http get sberbank url")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		zap.L().Warn(
			"requestForDeposit: read body",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed read body response from sberbank")
	}
	err = json.Unmarshal(body, &sr)
	if err != nil {
		zap.L().Warn(
			"requestForDeposit: unmarshal response by sberbank",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed unmarshal response from sberbank")
	}
	switch sr.ErrorCode {
	case "":
	case "0":
	default:
		return errors.New(sr.ErrorMessage)
	}
	return nil
}

// Отмена холдирования.
func (p *Provider) ReverseForHold(orderID string, amount int64) error {
	_url, _ := url.Parse(p.cfg.EntrypointURL + "/payment/rest/reverse.do")
	q := _url.Query()
	q.Add("orderId", orderID)
	q.Add("amount", strconv.FormatInt(amount, 10))
	q.Add("password", p.cfg.Password)
	q.Add("userName", p.cfg.UserName)
	_url.RawQuery = q.Encode()

	var sr SberbankResp
	res, err := http.Get(_url.String())
	if err != nil {
		zap.L().Warn(
			"reverse: get url",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed http get sberbank url")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		zap.L().Warn(
			"requestForReverse: read body",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed read body response from sberbank")
	}
	err = json.Unmarshal(body, &sr)
	if err != nil {
		zap.L().Warn(
			"requestForReverse: unmarshal response by sberbank",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed unmarshal response from sberbank")
	}
	switch sr.ErrorCode {
	case "":
	case "0":
	default:
		return errors.New(sr.ErrorMessage)
	}
	return nil
}

// Отмена списанных средств
func (p *Provider) Refund(orderID string, amount int64) error {
	_url, _ := url.Parse(p.cfg.EntrypointURL + "/payment/rest/refund.do")
	q := _url.Query()
	q.Add("orderId", orderID)
	q.Add("amount", strconv.FormatInt(amount, 10))
	q.Add("password", p.cfg.Password)
	q.Add("userName", p.cfg.UserName)
	_url.RawQuery = q.Encode()

	var sr SberbankResp
	res, err := http.Get(_url.String())
	if err != nil {
		zap.L().Warn(
			"refund: get url",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed http get sberbank url")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		zap.L().Warn(
			"requestForRefund: read body",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed read body response from sberbank")
	}
	err = json.Unmarshal(body, &sr)
	if err != nil {
		zap.L().Warn(
			"requestForRefund: unmarshal response by sberbank",
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed unmarshal response from sberbank")
	}
	switch sr.ErrorCode {
	case "":
	case "0":
	default:
		return errors.New(sr.ErrorMessage)
	}
	return nil
}

func (p *Provider) GetOrderRawStatus(orderID string) (*SberbankOrderStatus, error) {
	_url, _ := url.Parse(p.cfg.EntrypointURL + "/payment/rest/getOrderStatusExtended.do")
	q := _url.Query()
	q.Add("orderId", orderID)
	q.Add("token", p.cfg.Token)
	_url.RawQuery = q.Encode()

	var os SberbankOrderStatus
	res, err := http.Get(_url.String())
	if err != nil {
		p.l.Warn(
			"orderStatus: get url",
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http get sberbank url")
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		p.l.Warn(
			"requestForOrderStatus: read body",
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed read body response from sberbank")
	}
	err = json.Unmarshal(body, &os)
	if err != nil {
		p.l.Warn(
			"register: bad unmarshal request from sberbank",
			zap.String("body", string(body)),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed unmarshal response from sberbank")
	}
	return &os, nil
}

type Config struct {
	EntrypointURL string
	Token         string
	Password      string
	UserName      string
}

// Информация о переводе
type TransferInformation struct {
	ReturnURL   string
	FailURL     string
	Description string
	Email       string
}

func newPaymentOrderID() string {
	b := make([]byte, 3)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	tm := time.Now()
	return fmt.Sprintf(
		"app-%d-%d-%d-%d-%d-%d-%s",
		tm.Year(),
		tm.Month(),
		tm.Day(),
		tm.Hour(),
		tm.Minute(),
		tm.Second(),
		hex.EncodeToString(b))
}
