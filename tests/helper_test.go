package tests

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine/strategies/invoices/refund"
	isimple "github.com/gebv/acca/engine/strategies/invoices/simple"
	"github.com/gebv/acca/engine/strategies/transactions/moedelo"
	"github.com/gebv/acca/engine/strategies/transactions/sberbank"
	"github.com/gebv/acca/engine/strategies/transactions/sberbank_refund"
	tsimple "github.com/gebv/acca/engine/strategies/transactions/simple"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

type helperData struct {
	accC             api.AccountsClient
	invC             api.InvoicesClient
	authCtx          context.Context
	accIDs           map[string]int64
	balances         map[int64]int64
	acceptedBalances map[int64]int64
	invoiceIDs       map[string]int64
	transactionIDs   map[string]int64
	txProviderUrls   map[string]string
	txProviderIDs    map[string]string
	istrategies      map[string]string
	tstrategies      map[string]string
}

func NewHelperData() *helperData {

	h := helperData{
		accC: api.NewAccountsClient(Conn),
		invC: api.NewInvoicesClient(Conn),
		authCtx: metadata.NewOutgoingContext(Ctx, metadata.New(map[string]string{
			accessTokenMDKey: AccessToken,
		})),
		accIDs:           make(map[string]int64),
		balances:         make(map[int64]int64),
		acceptedBalances: make(map[int64]int64),
		invoiceIDs:       make(map[string]int64),
		transactionIDs:   make(map[string]int64),
		txProviderUrls:   make(map[string]string),
		txProviderIDs:    make(map[string]string),
		istrategies: map[string]string{
			"simple": new(isimple.Strategy).Name().String(),
			"refund": new(refund.Strategy).Name().String(),
		},
		tstrategies: map[string]string{
			"simple":          new(tsimple.Strategy).Name().String(),
			"sberbank":        new(sberbank.Strategy).Name().String(),
			"sberbank_refund": new(sberbank_refund.Strategy).Name().String(),
			"moedelo":         new(moedelo.Strategy).Name().String(),
		},
	}
	return &h
}

func (h *helperData) Sleep(s int) {
	time.Sleep(time.Duration(s) * time.Second)
}

func (h *helperData) CreateCurrency(key string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("CreateCurrency", func(t *testing.T) {
			_, err := h.accC.CreateCurrency(h.authCtx, &api.CreateCurrencyRequest{
				Key: key,
			})
			require.NoError(t, err)
		})
		t.Run("GetCurrency", func(t *testing.T) {
			res, err := h.accC.GetCurrency(h.authCtx, &api.GetCurrencyRequest{
				Key: key,
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetCurrency())
		})
	}
}

func (h *helperData) CreateAccount(accKey, currKey string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("CreateAccount", func(t *testing.T) {
			res, err := h.accC.CreateAccount(h.authCtx, &api.CreateAccountRequest{
				Key:         accKey,
				CurrencyKey: currKey,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res)
			h.accIDs[accKey] = res.GetAccId()
		})
		t.Run("GetAccountByKey", func(t *testing.T) {
			res, err := h.accC.GetAccountByKey(h.authCtx, &api.GetAccountByKeyRequest{
				Key:     accKey,
				CurrKey: currKey,
			})
			require.NoError(t, err)
			require.NotNil(t, res.GetAccount())
			require.EqualValues(t, h.accIDs[accKey], res.GetAccount().GetAccId())
			h.balances[h.accIDs[accKey]] = res.GetAccount().GetBalance()
			h.acceptedBalances[h.accIDs[accKey]] = res.GetAccount().GetBalanceAccepted()
		})
	}
}

func (h *helperData) GetInvoiceID(key string) int64 {
	return h.invoiceIDs[key]
}

func (h *helperData) GetTxID(key string) int64 {
	return h.transactionIDs[key]
}

func (h *helperData) NewInvoice(key, strategy string, meta *[]byte) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("NewInvoice", func(t *testing.T) {
			res, err := h.invC.NewInvoice(h.authCtx, &api.NewInvoiceRequest{
				Key:      key,
				Meta:     meta,
				Strategy: h.istrategies[strategy],
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoiceId())
			h.invoiceIDs[key] = res.GetInvoiceId()
		})
		t.Run("GetInvoiceByID", func(t *testing.T) {
			res, err := h.invC.GetInvoiceByIDs(h.authCtx, &api.GetInvoiceByIDsRequest{
				InvoiceIds: []int64{h.invoiceIDs[key]},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetInvoices())
			require.EqualValues(t, api.InvoiceStatus_DRAFT_I, res.GetInvoices()[0].GetStatus())
		})
	}
}

func (h *helperData) CreateOperation(
	srcAccKey, dstAccKey, holdAccKey string,
	hold bool,
	strategy api.OperStrategy,
	amount int64,
) *api.AddTransactionToInvoiceRequest_Oper {
	var holdAccID *int64
	if hold && h.accIDs[holdAccKey] != 0 {
		id := h.accIDs[holdAccKey]
		holdAccID = &id
	}
	return &api.AddTransactionToInvoiceRequest_Oper{
		SrcAccId:  h.accIDs[srcAccKey],
		DstAccId:  h.accIDs[dstAccKey],
		Strategy:  strategy,
		Amount:    amount,
		Meta:      nil,
		Hold:      hold,
		HoldAccId: holdAccID,
	}
}
func (h *helperData) AddTransactionToInvoice(
	invKey, txKey, strategy string,
	amount int64,
	meta *[]byte,
	opers []*api.AddTransactionToInvoiceRequest_Oper,
) func(t *testing.T) {
	return func(t *testing.T) {
		t.Run("AddTransactionToInvoice", func(t *testing.T) {
			res, err := h.invC.AddTransactionToInvoice(h.authCtx, &api.AddTransactionToInvoiceRequest{
				InvoiceId:  h.invoiceIDs[invKey],
				Key:        &txKey,
				Strategy:   h.tstrategies[strategy],
				Amount:     amount,
				Meta:       meta,
				Operations: opers,
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTxId())
			h.transactionIDs[txKey] = res.GetTxId()
		})

		t.Run("GetTransactionByID", func(t *testing.T) {
			res, err := h.invC.GetTransactionByIDs(h.authCtx, &api.GetTransactionByIDsRequest{
				TxIds: []int64{h.transactionIDs[txKey]},
			})
			require.NoError(t, err)
			require.NotEmpty(t, res.GetTransactions())
			require.EqualValues(t, api.TxStatus_DRAFT_TX, res.GetTransactions()[0].GetStatus())
		})
	}
}
func (h *helperData) BalanceDec(accKey string, amount int64) {
	h.balances[h.accIDs[accKey]] -= amount
}

func (h *helperData) AcceptedBalanceDec(accKey string, amount int64) {
	h.acceptedBalances[h.accIDs[accKey]] -= amount
}

func (h *helperData) BalanceInc(accKey string, amount int64) {
	h.balances[h.accIDs[accKey]] += amount
}

func (h *helperData) AcceptedBalanceInc(accKey string, amount int64) {
	h.acceptedBalances[h.accIDs[accKey]] += amount
}

func (h *helperData) AuthInvoice(invKey string) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := h.invC.AuthInvoice(h.authCtx, &api.AuthInvoiceRequest{
			InvoiceId: h.invoiceIDs[invKey],
		})
		require.NoError(t, err)
	}
}

func (h *helperData) AcceptInvoice(invKey string) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := h.invC.AcceptInvoice(h.authCtx, &api.AcceptInvoiceRequest{
			InvoiceId: h.invoiceIDs[invKey],
		})
		require.NoError(t, err)
	}
}

func (h *helperData) RejectInvoice(invKey string) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := h.invC.RejectInvoice(h.authCtx, &api.RejectInvoiceRequest{
			InvoiceId: h.invoiceIDs[invKey],
		})
		require.NoError(t, err)
	}
}

func (h *helperData) CheckBalances(accKey, currKey string) func(t *testing.T) {
	return func(t *testing.T) {
		res, err := h.accC.GetAccountByKey(h.authCtx, &api.GetAccountByKeyRequest{
			Key:     accKey,
			CurrKey: currKey,
		})
		require.NoError(t, err)
		require.NotNil(t, res.GetAccount())
		require.EqualValues(t, h.accIDs[accKey], res.GetAccount().GetAccId())
		require.EqualValues(t, h.balances[h.accIDs[accKey]], res.GetAccount().GetBalance())
		require.EqualValues(t, h.acceptedBalances[h.accIDs[accKey]], res.GetAccount().GetBalanceAccepted())
	}
}

func (h *helperData) CheckTransactionWithProvider(txKey, providerStatus string, txStatus api.TxStatus) func(t *testing.T) {
	return func(t *testing.T) {
		res, err := h.invC.GetTransactionByIDs(h.authCtx, &api.GetTransactionByIDsRequest{
			TxIds: []int64{h.transactionIDs[txKey]},
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.GetTransactions())
		require.EqualValues(t, txStatus, res.GetTransactions()[0].GetStatus())
		require.NotNil(t, res.GetTransactions()[0].GetProviderOperStatus())
		require.EqualValues(t, providerStatus, *res.GetTransactions()[0].GetProviderOperStatus())
		require.NotNil(t, res.GetTransactions()[0].GetProviderOperUrl())
		h.txProviderUrls[txKey] = *res.GetTransactions()[0].GetProviderOperUrl()
		require.NotNil(t, res.GetTransactions()[0].GetProviderOperId())
		h.txProviderIDs[txKey] = *res.GetTransactions()[0].GetProviderOperId()
	}
}

func (h *helperData) GetTxProviderID(txKey string) string {
	return h.txProviderIDs[txKey]
}

func (h *helperData) CheckTransaction(txKey string, txStatus api.TxStatus) func(t *testing.T) {
	return func(t *testing.T) {
		res, err := h.invC.GetTransactionByIDs(h.authCtx, &api.GetTransactionByIDsRequest{
			TxIds: []int64{h.transactionIDs[txKey]},
		})
		require.NoError(t, err)
		require.NotEmpty(t, res.GetTransactions())
		require.EqualValues(t, txStatus, res.GetTransactions()[0].GetStatus())
	}
}

func (h *helperData) BalanceChanges(accKey string) func(t *testing.T) {
	return func(t *testing.T) {
		accID := h.accIDs[accKey]
		res, err := h.accC.BalanceChanges(h.authCtx, &api.BalanceChangesRequest{
			Offset: 0,
			Limit:  1,
			AccId:  &accID,
		})
		require.NoError(t, err)
		require.Len(t, res.GetBalanceChanges(), 1)
		require.EqualValues(t, h.accIDs[accKey], res.GetBalanceChanges()[0].GetAccId())
		require.EqualValues(t, h.balances[h.accIDs[accKey]], res.GetBalanceChanges()[0].GetBalance())
		require.EqualValues(t, h.acceptedBalances[h.accIDs[accKey]], res.GetBalanceChanges()[0].GetBalanceAccepted())
	}
}

func (h *helperData) AuthTx(txKey string) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := h.invC.AuthTx(h.authCtx, &api.AuthTxRequest{
			TxId: h.transactionIDs[txKey],
		})
		require.NoError(t, err)
	}
}

func (h *helperData) AcceptTx(txKey string) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := h.invC.AcceptTx(h.authCtx, &api.AcceptTxRequest{
			TxId: h.transactionIDs[txKey],
		})
		require.NoError(t, err)
	}
}

func (h *helperData) RejectTx(txKey string) func(t *testing.T) {
	return func(t *testing.T) {
		_, err := h.invC.RejectTx(h.authCtx, &api.RejectTxRequest{
			TxId: h.transactionIDs[txKey],
		})
		require.NoError(t, err)
	}
}

func (h *helperData) SendCardDataInSberbank(txKey string) func(t *testing.T) {
	// https://3dsec.sberbank.ru/payment/merchants/sbersafe/payment_ru.html?mdOrder=ebc0d85c-42e9-7593-96af-650104b2e43b
	return func(t *testing.T) {
		URL := h.txProviderUrls[txKey]
		if len(URL) < 37 {
			t.Fatal("URL: ", URL)
		}
		c := http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		}
		resp, err := c.Post(
			"https://3dsec.sberbank.ru/payment/rest/processform.do?MDORDER="+
				URL[len(URL)-36:]+
				"&$PAN=5555555555555599&$CVC=123&MM=12&YYYY=2019&language=ru&TEXT=CARDHOLDER+NAME",
			"",
			nil,
		)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		errCode := &struct {
			ErrorCode int64  `json:"errorCode"`
			Redirect  string `json:"redirect"`
		}{}
		err = json.Unmarshal(body, errCode)
		require.NoError(t, err)
		require.EqualValues(t, 0, errCode.ErrorCode)
		h.getSberbankWebhook(t, errCode.Redirect)
	}
}

func (h *helperData) getSberbankWebhook(t *testing.T, URL string) {
	t.Run("GetSberbankWebhook", func(t *testing.T) {
		c := http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		}
		t.Log("URL: ", URL)
		resp, err := c.Get("http://localhost:10003/webhook/sberbank" + URL[len("localhost"):])
		require.NoError(t, err)
		defer resp.Body.Close()
	})
}
