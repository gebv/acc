package tests

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine/strategies/invoices/refund"
	isimple "github.com/gebv/acca/engine/strategies/invoices/simple"
	"github.com/gebv/acca/engine/strategies/transactions/moedelo"
	"github.com/gebv/acca/engine/strategies/transactions/sberbank"
	"github.com/gebv/acca/engine/strategies/transactions/sberbank_refund"
	tsimple "github.com/gebv/acca/engine/strategies/transactions/simple"
	"github.com/gebv/acca/engine/strategies/transactions/stripe"
	"github.com/gebv/acca/engine/strategies/transactions/stripe_refund"
)

type helperData struct {
	rw               sync.RWMutex
	accU             api.UpdatesClient
	update           api.Updates_GetUpdateClient
	updates          []*api.Update
	updateCh         chan *api.Update
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

func NewHelperData(t *testing.T) *helperData {
	h := helperData{
		accU:     api.NewUpdatesClient(Conn),
		updates:  make([]*api.Update, 0, 100),
		updateCh: make(chan *api.Update, 100),
		accC:     api.NewAccountsClient(Conn),
		invC:     api.NewInvoicesClient(Conn),
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
			"stripe":          new(stripe.Strategy).Name().String(),
			"stripe_refund":   new(stripe_refund.Strategy).Name().String(),
		},
	}
	res, err := h.accU.GetUpdate(h.authCtx, &api.GetUpdateRequest{})
	require.NoError(t, err)
	h.update = res
	go func() {
		for {
			u, err := h.update.Recv()
			if err != nil {
				return
			}
			h.rw.Lock()
			h.updates = append(h.updates, u)
			h.rw.Unlock()
			h.updateCh <- u
		}
	}()
	return &h
}

func (h *helperData) CompareUpdates(updates []*api.Update) func(t *testing.T) {
	return func(t *testing.T) {
		h.rw.RLock()
		defer h.rw.RUnlock()
		require.Len(t, h.updates, len(updates))
		for i, u := range updates {
			if u.GetUpdatedInvoice() != nil {
				require.EqualValues(
					t,
					u.GetUpdatedInvoice().GetInvoiceId(),
					h.updates[i].GetUpdatedInvoice().GetInvoiceId(),
				)
				require.EqualValues(
					t,
					u.GetUpdatedInvoice().GetStatus(),
					h.updates[i].GetUpdatedInvoice().GetStatus(),
				)
			} else {
				require.EqualValues(
					t,
					u.GetUpdatedTransaction().GetTransactionId(),
					h.updates[i].GetUpdatedTransaction().GetTransactionId(),
				)
				require.EqualValues(
					t,
					u.GetUpdatedTransaction().GetStatus(),
					h.updates[i].GetUpdatedTransaction().GetStatus(),
				)
			}
		}
	}
}

func (h *helperData) WaitInvoice(invKey string, invStatus api.InvoiceStatus) func(t *testing.T) {
	return func(t *testing.T) {
		timer := time.NewTimer(33 * time.Second)
		defer timer.Stop()
		for {
			select {
			case u := <-h.updateCh:
				if u.GetUpdatedInvoice().GetInvoiceId() == h.invoiceIDs[invKey] &&
					u.GetUpdatedInvoice().GetStatus() == invStatus {
					return
				}
			case <-timer.C:
				t.Error("timeout")
				return
			}
		}
	}
}

func (h *helperData) WaitTransaction(txKey string, txStatus api.TxStatus) func(t *testing.T) {
	return func(t *testing.T) {
		timer := time.NewTimer(33 * time.Second)
		defer timer.Stop()
		for {
			select {
			case u := <-h.updateCh:
				if u.GetUpdatedTransaction().GetTransactionId() == h.transactionIDs[txKey] &&
					u.GetUpdatedTransaction().GetStatus() == txStatus {
					return
				}
			case <-timer.C:
				t.Error("timeout")
				return
			}
		}
	}
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

func (h *helperData) SendCardDataInStripe(txKey string) func(t *testing.T) {
	return func(t *testing.T) {
		paymentIntentID := h.txProviderIDs[txKey]
		clientSecret := h.txProviderUrls[txKey]
		c := http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		}
		reqBody := strings.NewReader(`payment_method_data[type]=card&payment_method_data[card][number]=4242424242424242&payment_method_data[card][cvc]=242&payment_method_data[card][exp_month]=04&payment_method_data[card][exp_year]=24&payment_method_data[billing_details][address][postal_code]=42442&payment_method_data[guid]=9eda6b0a-59a2-4229-9e5e-98d39e75b16a&payment_method_data[muid]=22f2306d-8ae1-49d3-8d74-1bcf5fe6bfa0&payment_method_data[sid]=66e15d10-c341-4a01-b3db-4482563a46e6&payment_method_data[payment_user_agent]=stripe.js%2Fffcf9782%3B+stripe-js-v3%2Fffcf9782&payment_method_data[referrer]=http%3A%2F%2Flocalhost%3A8080%2Fstatic%2Ftest2.html&expected_payment_method_type=card&use_stripe_sdk=true&key=pk_test_Ij2QA1jVfWPxLHWl6WoZL91Y00XKOGIOoy&client_secret=` + clientSecret)
		req, err := http.NewRequest("POST", "https://api.stripe.com/v1/payment_intents/"+paymentIntentID+"/confirm", reqBody)
		if err != nil {
			// handle err
		}
		req.Host = "api.stripe.com"
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://js.stripe.com")
		req.Header.Set("Content-Length", "794")
		req.Header.Set("Accept-Language", "ru")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.2 Safari/605.1.15")
		req.Header.Set("Referer", "https://js.stripe.com/")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Connection", "keep-alive")

		resp, err := c.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		log.Println("BODY: ", string(body))

		// BODY
		// {
		//  "id": "pi_1FayQ8Bz5RLqjsMcT8NCHfZP",
		//  "object": "payment_intent",
		//  "amount": 2214,
		//  "canceled_at": null,
		//  "cancellation_reason": null,
		//  "capture_method": "automatic",
		//  "client_secret": "pi_1FayQ8Bz5RLqjsMcT8NCHfZP_secret_GEY1IFAGtTYvSRMl4DxE6dz7a",
		//  "confirmation_method": "automatic",
		//  "created": 1572846488,
		//  "currency": "usd",
		//  "description": null,
		//  "last_payment_error": null,
		//  "livemode": false,
		//  "next_action": null,
		//  "payment_method": "pm_1FayRDBz5RLqjsMc92YaLRg3",
		//  "payment_method_types": [
		//    "card"
		//  ],
		//  "receipt_email": null,
		//  "setup_future_usage": null,
		//  "shipping": null,
		//  "source": null,
		//  "status": "succeeded"
		//}
		//err = json.Unmarshal(body, errCode)
		//require.NoError(t, err)
		//require.EqualValues(t, 0, errCode.ErrorCode)

	}
}

func (h *helperData) SendConfirmWithPaymentMethodInStripe(txKey string) func(t *testing.T) {
	return func(t *testing.T) {
		paymentIntentID := h.txProviderIDs[txKey]
		clientSecret := h.txProviderUrls[txKey]
		c := http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		}
		reqBody := strings.NewReader(`payment_method=pm_1FbHHfBz5RLqjsMcdqPUfpaE&expected_payment_method_type=card&use_stripe_sdk=true&key=pk_test_Ij2QA1jVfWPxLHWl6WoZL91Y00XKOGIOoy&client_secret=` + clientSecret)
		req, err := http.NewRequest("POST", "https://api.stripe.com/v1/payment_intents/"+paymentIntentID+"/confirm", reqBody)
		if err != nil {
			// handle err
		}
		req.Host = "api.stripe.com"
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://js.stripe.com")
		req.Header.Set("Content-Length", "794")
		req.Header.Set("Accept-Language", "ru")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.2 Safari/605.1.15")
		req.Header.Set("Referer", "https://js.stripe.com/")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Connection", "keep-alive")

		resp, err := c.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		log.Println("BODY: ", string(body))

	}
}

func (h *helperData) SendConfirmPaymentInStripe(txKey string) func(t *testing.T) {
	return func(t *testing.T) {
		paymentIntentID := h.txProviderIDs[txKey]
		clientSecret := h.txProviderUrls[txKey]
		c := http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		}
		reqBody := strings.NewReader(`key=pk_test_Ij2QA1jVfWPxLHWl6WoZL91Y00XKOGIOoy&client_secret=` + clientSecret)
		req, err := http.NewRequest("POST", "https://api.stripe.com/v1/payment_intents/"+paymentIntentID+"/confirm", reqBody)
		if err != nil {
			// handle err
		}
		req.Host = "api.stripe.com"
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Origin", "https://js.stripe.com")
		req.Header.Set("Content-Length", "794")
		req.Header.Set("Accept-Language", "ru")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.2 Safari/605.1.15")
		req.Header.Set("Referer", "https://js.stripe.com/")
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		req.Header.Set("Connection", "keep-alive")

		resp, err := c.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		require.NoError(t, err)
		log.Println("BODY: ", string(body))

	}
}
