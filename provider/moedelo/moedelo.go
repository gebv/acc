package moedelo

import (
	"net/url"
	"strconv"
	"time"

	"github.com/gebv/acca/provider"
	"github.com/nats-io/nats.go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/reform.v1"
)

const (
	MOEDELO provider.Provider = "moedelo"
)

var (
	ErrProviderNotSet = errors.New("Provider not set")
)

const TimeFormat = "2006-01-02T15:04:05-07:00"

func NewProvider(db *reform.DB, cfg Config, nc *nats.EncodedConn) *Provider {
	var c *client
	if cfg.Token != "" {
		c = newClient(cfg.Token)
	}
	return &Provider{
		db:  db,
		nc:  nc,
		cfg: cfg,
		c:   c,
		l:   zap.L().Named("moedelo_provider"),
	}
}

type Provider struct {
	cfg Config
	db  *reform.DB
	nc  *nats.EncodedConn
	c   *client
	l   *zap.Logger
}

type Config struct {
	EntrypointURL string
	Token         string
}

// CreateKontragent создает контрагента в моем деле.
// Возвращает числовой идентификатор контрагента.
func (p *Provider) CreateKontragent(
	inn string,
	name string,
) (*int64, error) {
	if p.c == nil {
		return nil, ErrProviderNotSet
	}
	_url, err := url.Parse(p.cfg.EntrypointURL + "/kontragents/api/v1/kontragent")
	if err != nil {
		return nil, errors.Wrap(err, "Failed parse url")
	}
	in := &KontragentModel{
		Inn:  inn,
		Name: name,
		Type: Kontragent, // TODO уточнить
		Form: UL,         // TODO уточнить
		// TODO уточнить указание адресса контрагента
	}
	out := &KontragentRepresentation{}
	err = p.c.POSTAndUnmarshalJson(_url.String(), in, out)
	if err != nil {
		p.l.Warn(
			"create kontragent",
			zap.String("url", _url.String()),
			zap.Any("in", in),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http post request")
	}

	return &out.ID, nil
}

// CreateAccount создает счета контрагента в моем деле.
// Возвращает числовой идентификатор счета.
func (p *Provider) CreateAccount(
	kontragentID int64,
	bik string,
	number string,
	comment string,
) (*int64, error) {
	if p.c == nil {
		return nil, ErrProviderNotSet
	}
	_url, err := url.Parse(
		p.cfg.EntrypointURL +
			"/kontragents/api/v1/kontragent/" +
			strconv.FormatInt(kontragentID, 10) +
			"/account")
	if err != nil {
		return nil, errors.Wrap(err, "Failed parse url")
	}
	in := &KontragentSettlementAccountModel{
		Bik:     bik,
		Number:  number,
		Comment: comment,
	}
	out := &KontragentSettlementAccountRepresentation{}
	err = p.c.POSTAndUnmarshalJson(_url.String(), in, out)
	if err != nil {
		p.l.Warn(
			"create account from kontragent",
			zap.String("url", _url.String()),
			zap.Any("in", in),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http post request")
	}

	return &out.ID, nil
}

// UpdateAccount обновляет счет контрагента в моем деле.
func (p *Provider) UpdateAccount(
	kontragentID int64,
	accountID int64,
	bik string,
	number string,
	comment string,
) error {
	if p.c == nil {
		return ErrProviderNotSet
	}
	_url, err := url.Parse(
		p.cfg.EntrypointURL +
			"/kontragents/api/v1/kontragent/" +
			strconv.FormatInt(kontragentID, 10) +
			"/account/" +
			strconv.FormatInt(accountID, 10))
	if err != nil {
		return errors.Wrap(err, "Failed parse url")
	}

	in := &KontragentSettlementAccountModel{
		Bik:     bik,
		Number:  number,
		Comment: comment,
	}
	out := &KontragentSettlementAccountRepresentation{}
	err = p.c.PUTAndUnmarshalJson(_url.String(), in, out)
	if err != nil {
		p.l.Warn(
			"update account from kontragent",
			zap.String("url", _url.String()),
			zap.Any("in", in),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed http post request")
	}

	return nil
}

// UpdateKontragent обновляет контрагента в моем деле.
func (p *Provider) UpdateKontragent(
	kontragentID int64,
	inn string,
	name string,
) error {
	if p.c == nil {
		return ErrProviderNotSet
	}
	_url, err := url.Parse(
		p.cfg.EntrypointURL +
			"/kontragents/api/v1/kontragent/" +
			strconv.FormatInt(kontragentID, 10))
	if err != nil {
		return errors.Wrap(err, "Failed parse url")
	}

	in := &KontragentModel{
		Inn:  inn,
		Name: name,
		Type: Kontragent, // TODO уточнить
		Form: UL,         // TODO уточнить
		// TODO уточнить указание адресса контрагента
	}
	out := &KontragentRepresentation{}
	err = p.c.PUTAndUnmarshalJson(_url.String(), in, out)
	if err != nil {
		p.l.Warn(
			"update kontragent",
			zap.String("url", _url.String()),
			zap.Any("in", in),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed http post request")
	}
	return nil
}

// GetAccount возвращает счет контрагента из моего дела.
func (p *Provider) GetAccount(
	kontragentID int64,
	accountID int64,
) (*KontragentSettlementAccountRepresentation, error) {
	if p.c == nil {
		return nil, ErrProviderNotSet
	}
	_url, err := url.Parse(
		p.cfg.EntrypointURL +
			"/kontragents/api/v1/kontragent/" +
			strconv.FormatInt(kontragentID, 10) +
			"/account/" +
			strconv.FormatInt(accountID, 10))
	if err != nil {
		return nil, errors.Wrap(err, "Failed parse url")
	}
	out := &KontragentSettlementAccountRepresentation{}
	err = p.c.GETAndUnmarshalJson(_url.String(), out)
	if err != nil {
		p.l.Warn(
			"get account from kontragent",
			zap.String("url", _url.String()),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http get request")
	}
	return out, nil
}

// GetKontragent возвращает контрагента из моего дела.
func (p *Provider) GetKontragent(
	kontragentID int64,
) (*KontragentRepresentation, error) {
	if p.c == nil {
		return nil, ErrProviderNotSet
	}
	_url, err := url.Parse(
		p.cfg.EntrypointURL +
			"/kontragents/api/v1/kontragent/" +
			strconv.FormatInt(kontragentID, 10))
	if err != nil {
		return nil, errors.Wrap(err, "Failed parse url")
	}
	out := &KontragentRepresentation{}
	err = p.c.GETAndUnmarshalJson(_url.String(), out)
	if err != nil {
		p.l.Warn(
			"get kontragent",
			zap.String("url", _url.String()),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http get request")
	}

	return out, nil
}

// CreateBill создает счета в моем деле.
// Возвращает числовой идентификатор счета и ссылку.
func (p *Provider) CreateBill(
	kontragentID int64,
	docDate time.Time,
	items []SalesDocumentItemModel,
) (*int64, *string, error) {
	if p.c == nil {
		return nil, nil, ErrProviderNotSet
	}
	_url, err := url.Parse(p.cfg.EntrypointURL + "/accounting/api/v1/sales/bill")
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed parse url")
	}
	in := &BillSaveRequestModel{
		DocDate:      docDate.Format(TimeFormat),
		Type:         Usual,
		KontragentID: kontragentID,
		Items:        items,
	}
	out := &BillRepresentation{}
	err = p.c.POSTAndUnmarshalJson(_url.String(), in, out)
	if err != nil {
		p.l.Warn(
			"create bill",
			zap.String("url", _url.String()),
			zap.Any("in", in),
			zap.Error(err),
		)
		return nil, nil, errors.Wrap(err, "Failed http post request")
	}
	link := "https://moedelo.org" + out.Online
	return &out.ID, &link, nil
}

// UpdateBill создает счета в моем деле.
// Возвращает числовой идентификатор счета и ссылку.
func (p *Provider) UpdateBill(
	billID int64,
	kontragentID int64,
	docDate time.Time,
	items []SalesDocumentItemModel,
	status *BillStatus,
) error {
	if p.c == nil {
		return ErrProviderNotSet
	}
	_url, err := url.Parse(p.cfg.EntrypointURL + "/accounting/api/v1/sales/bill/" + strconv.FormatInt(billID, 10))
	if err != nil {
		return errors.Wrap(err, "Failed parse url")
	}

	in := &BillSaveRequestModel{
		DocDate:      docDate.Format(TimeFormat),
		Type:         Usual,
		KontragentID: kontragentID,
		Items:        items,
	}
	if status != nil {
		in.Status = *status
	}
	out := &BillRepresentation{}
	err = p.c.PUTAndUnmarshalJson(_url.String(), in, out)
	if err != nil {
		p.l.Warn(
			"update bill",
			zap.String("url", _url.String()),
			zap.Any("in", in),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed http post request")
	}
	return nil
}

// GetBill возвращает счет из моего дела.
func (p *Provider) GetBill(
	billID int64,
) (*BillRepresentation, error) {
	if p.c == nil {
		return nil, ErrProviderNotSet
	}
	_url, err := url.Parse(
		p.cfg.EntrypointURL +
			"/accounting/api/v1/sales/bill/" +
			strconv.FormatInt(billID, 10))
	if err != nil {
		return nil, errors.Wrap(err, "Failed parse url")
	}
	out := &BillRepresentation{}
	err = p.c.GETAndUnmarshalJson(_url.String(), out)
	if err != nil {
		p.l.Warn(
			"get bill",
			zap.String("url", _url.String()),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http get request")
	}
	return out, nil
}

// GetListBills возвращает список счетов из моего дела по датам.
func (p *Provider) GetListBills(
	afterDate *time.Time,
	beforeDate *time.Time,
) (*BillRepresentationCollection, error) {
	if p.c == nil {
		return nil, ErrProviderNotSet
	}
	_url, err := url.Parse(p.cfg.EntrypointURL + "/accounting/api/v1/sales/bill")
	if err != nil {
		return nil, errors.Wrap(err, "Failed parse url")
	}
	q := _url.Query()
	if afterDate != nil {
		q.Add("afterDate", afterDate.Format(TimeFormat))
	}
	if beforeDate != nil {
		q.Add("beforeDate", beforeDate.Format(TimeFormat))
	}
	_url.RawQuery = q.Encode()

	out := &BillRepresentationCollection{}
	err = p.c.GETAndUnmarshalJson(_url.String(), out)
	if err != nil {
		p.l.Warn(
			"get bill",
			zap.String("url", _url.String()),
			zap.Error(err),
		)
		return nil, errors.Wrap(err, "Failed http get request")
	}
	return out, nil
}

// DeleteBill удаляет счет из моего дела.
func (p *Provider) DeleteBill(
	billID int64,
) error {
	if p.c == nil {
		return ErrProviderNotSet
	}
	_url, err := url.Parse(
		p.cfg.EntrypointURL +
			"/accounting/api/v1/sales/bill/" +
			strconv.FormatInt(billID, 10))
	if err != nil {
		return errors.Wrap(err, "Failed parse url")
	}
	out := &BillRepresentation{}
	err = p.c.GETAndUnmarshalJson(_url.String(), out)
	if err != nil {
		p.l.Warn(
			"delete bill",
			zap.String("url", _url.String()),
			zap.Error(err),
		)
		return errors.Wrap(err, "Failed http get request")
	}
	return nil
}
