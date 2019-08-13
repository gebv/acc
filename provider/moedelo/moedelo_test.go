package moedelo

import (
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func zzzTestProviderServiceMoedelo(t *testing.T) {
	config := zap.NewDevelopmentConfig()
	config.Level.SetLevel(zap.DebugLevel)
	l, err := config.Build(zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(l)
	zap.RedirectStdLog(l.Named("stdlog"))

	md := NewProvider(
		nil,
		Config{
			EntrypointURL: os.Getenv("MOEDELO_ENTRYPOINT_URL"),
			Token:         os.Getenv("MOEDELO_TOKEN"),
		},
		nil,
	)
	kontragentID, err := md.CreateKontragent("7720239606", "Иванов Иван Иванович")
	require.NoError(t, err)
	require.NotEmpty(t, kontragentID)
	accountID, err := md.CreateAccount(*kontragentID, "045655001", "40502810725000001126", "")
	require.NoError(t, err)
	require.NotEmpty(t, accountID)
	err = md.UpdateAccount(*kontragentID, *accountID, "045655001", "40502810725000001125", "")
	require.NoError(t, err)
	err = md.UpdateKontragent(*kontragentID, "7736207543", "Иванов Иван Иванович")
	require.NoError(t, err)
	kontragent, err := md.GetKontragent(*kontragentID)
	require.NoError(t, err)
	require.NotEmpty(t, kontragent)
	require.EqualValues(t, "7736207543", kontragent.Inn)
	account, err := md.GetAccount(*kontragentID, *accountID)
	require.NoError(t, err)
	require.NotEmpty(t, account)
	require.EqualValues(t, "40502810725000001125", account.Number)
	items := []SalesDocumentItemModel{
		{
			Name:    "Оплата заказа 1",
			Count:   1,
			Unit:    "шт",
			Type:    2,
			Price:   123321,
			NdsType: Nds0,
		},
		{
			Name:    "Оплата заказа 2",
			Count:   1,
			Unit:    "шт",
			Type:    2,
			Price:   321123,
			NdsType: Nds0,
		},
	}
	billID, billLink, err := md.CreateBill(*kontragentID, time.Now(), items)
	require.NoError(t, err)
	require.NotEmpty(t, billID)
	require.NotEmpty(t, billLink)
	log.Println("Bill Link: ", *billLink)
	items = []SalesDocumentItemModel{
		{
			Name:    "Оплата заказа 1",
			Count:   1,
			Unit:    "шт",
			Type:    Service,
			Price:   321123,
			NdsType: Nds0,
		},
		{
			Name:    "Оплата заказа 2",
			Count:   1,
			Unit:    "шт",
			Type:    Service,
			Price:   123321,
			NdsType: Nds0,
		},
	}
	err = md.UpdateBill(*billID, *kontragentID, time.Now(), items, nil)
	require.NoError(t, err)
	bill, err := md.GetBill(*billID)
	require.NoError(t, err)
	require.NotEmpty(t, bill)
	require.Len(t, bill.Items, 2)
	require.EqualValues(t, 321123.0, bill.Items[0].Price)
	require.EqualValues(t, 123321.0, bill.Items[1].Price)
}
