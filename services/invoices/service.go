package invoices

import (
	"context"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"gopkg.in/reform.v1"

	"github.com/gebv/acca/api"
	"github.com/gebv/acca/engine"
	"github.com/gebv/acca/engine/strategies"
	"github.com/gebv/acca/provider"
	"github.com/gebv/acca/provider/sberbank"
	"github.com/gebv/acca/services"
)

func NewServer(db *reform.DB, fs *firestore.Client) *Server {
	return &Server{
		db: db,
		fs: fs,
	}
}

type Server struct {
	db *reform.DB
	fs *firestore.Client
}

func (s Server) NewInvoice(ctx context.Context, req *api.NewInvoiceRequest) (*api.NewInvoiceResponse, error) {
	clientID := services.GetClient(ctx).ClientID

	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.StringAttribute("key", req.GetKey()),
		trace.StringAttribute("strategy", req.GetStrategy()),
	)

	key := strings.TrimSpace(strings.ToLower(req.GetKey()))
	strategy := strings.TrimSpace(strings.ToLower(req.GetStrategy()))

	if name := strategies.ExistInvName(strategy); name != strategies.UNDEFINED_INV {
		if str := strategies.GetInvoiceStrategy(name); str != nil && str.MetaValidation(req.GetMeta()) != nil {
			return nil, api.MakeError(codes.InvalidArgument, "Meta is not validate.")
		}
	} else {
		return nil, api.MakeError(codes.NotFound, "Strategy is not found.")
	}

	inv := &engine.Invoice{
		ClientID: &clientID,
		Key:      key,
		Strategy: strategy,
		Status:   engine.DRAFT_I,
		Meta:     req.GetMeta(),
		Payload:  nil,
	}
	if err := s.db.Insert(inv); err != nil {
		return nil, errors.Wrap(err, "failed insert new invoice")
	}
	return &api.NewInvoiceResponse{InvoiceId: inv.InvoiceID}, nil
}

func (s Server) GetInvoiceByIDs(ctx context.Context, req *api.GetInvoiceByIDsRequest) (*api.GetInvoiceByIDsResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	invoiceIDs := make([]string, 0, len(req.GetInvoiceIds()))
	for _, v := range req.GetInvoiceIds() {
		invoiceIDs = append(invoiceIDs, strconv.FormatInt(v, 10))
	}
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.StringAttribute("invoice_ids", strings.Join(invoiceIDs, ", ")),
	)
	if len(req.GetInvoiceIds()) == 0 {
		return &api.GetInvoiceByIDsResponse{}, nil
	}
	invoices := make([]*api.GetInvoiceByIDsResponse_Invoice, 0, len(req.GetInvoiceIds()))
	switch req.GetWithTx() {
	case true:
		list, err := s.db.SelectAllFrom(
			engine.ViewInvoiceTable,
			"WHERE client_id = $1 AND invoice_id = ANY ($2)",
			clientID,
			pq.Int64Array(req.GetInvoiceIds()),
		)
		if err != nil {
			return nil, errors.Wrap(err, "Failed get invoice by IDs.")
		}
		for _, v := range list {
			inv := v.(*engine.ViewInvoice)
			var nextStatus api.InvoiceStatus
			if inv.NextStatus != nil {
				nextStatus = MapInvStatusToApiInvStatus[*inv.NextStatus]
			}
			transactions := make([]*api.Tx, 0, len(inv.Transactions))
			for _, tr := range inv.Transactions {
				var nextStatusTr api.TxStatus
				if tr.NextStatus != nil {
					nextStatusTr = MapTrStatusToApiTrStatus[*tr.NextStatus]
				}
				opers := make([]*api.Oper, 0, len(tr.Operations))
				for _, op := range tr.Operations {
					opers = append(opers, &api.Oper{
						OperId:    op.OperationID,
						InvoiceId: op.InvoiceID,
						TxId:      op.TransactionID,
						SrcAccId:  op.SrcAccID,
						Hold:      op.Hold,
						HoldAccId: op.HoldAccID,
						DstAccId:  op.DstAccID,
						Strategy:  MapOperStrategyToApiTrStrategy[op.Strategy],
						Amount:    op.Amount,
						Key:       op.Key,
						Meta:      op.Meta,
						Status:    MapOperStatusToApiTrStatus[op.Status],
						CreatedAt: &op.CreatedAt,
						UpdatedAt: &op.UpdatedAt,
					})
				}
				var meta *[]byte
				if tr.Meta != nil {
					b := []byte(*tr.Meta)
					meta = &b
				}
				transactions = append(transactions, &api.Tx{
					TxId:               tr.TransactionID,
					InvoiceId:          tr.InvoiceID,
					Key:                tr.Key,
					Strategy:           tr.Strategy,
					Amount:             tr.Amount,
					Provider:           MapTrProviderToApiTrProvider[tr.Provider],
					ProviderOperId:     tr.ProviderOperID,
					ProviderOperStatus: tr.ProviderOperStatus,
					ProviderOperUrl:    tr.ProviderOperUrl,
					Meta:               meta,
					Status:             MapTrStatusToApiTrStatus[tr.Status],
					NextStatus:         nextStatusTr,
					CreatedAt:          &tr.CreatedAt,
					UpdatedAt:          &tr.UpdatedAt,
					Operations:         opers,
				})
			}
			invoices = append(invoices, &api.GetInvoiceByIDsResponse_Invoice{
				InvoiceId:    inv.InvoiceID,
				Key:          inv.Key,
				Status:       MapInvStatusToApiInvStatus[inv.Status],
				NextStatus:   nextStatus,
				Strategy:     inv.Strategy,
				Meta:         inv.Meta,
				CreatedAt:    &inv.CreatedAt,
				UpdatedAt:    &inv.UpdatedAt,
				Transactions: transactions,
			})
		}
	default:
		list, err := s.db.SelectAllFrom(
			engine.InvoiceTable,
			"WHERE client_id = $1 AND invoice_id = ANY ($2)",
			clientID,
			pq.Int64Array(req.GetInvoiceIds()),
		)
		if err != nil {
			return nil, errors.Wrap(err, "Failed get invoice by IDs.")
		}
		for _, v := range list {
			inv := v.(*engine.Invoice)
			var nextStatus api.InvoiceStatus
			if inv.NextStatus != nil {
				nextStatus = MapInvStatusToApiInvStatus[*inv.NextStatus]
			}
			invoices = append(invoices, &api.GetInvoiceByIDsResponse_Invoice{
				InvoiceId:  inv.InvoiceID,
				Key:        inv.Key,
				Status:     MapInvStatusToApiInvStatus[inv.Status],
				NextStatus: nextStatus,
				Strategy:   inv.Strategy,
				Meta:       inv.Meta,
				CreatedAt:  &inv.CreatedAt,
				UpdatedAt:  &inv.UpdatedAt,
			})
		}
	}
	return &api.GetInvoiceByIDsResponse{
		Invoices: invoices,
	}, nil
}

func (s Server) AddTransactionToInvoice(ctx context.Context, req *api.AddTransactionToInvoiceRequest) (*api.AddTransactionToInvoiceResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	var key string
	if req.GetKey() != nil {
		key = *req.GetKey()
	}
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.Int64Attribute("invoice_id", req.GetInvoiceId()),
		trace.StringAttribute("key", key),
		trace.StringAttribute("strategy", req.GetStrategy()),
		trace.Int64Attribute("amount", req.GetAmount()),
	)
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	if invoice.ClientID == nil || *invoice.ClientID != clientID {
		return nil, api.MakeError(codes.NotFound, "Invoice is not found.")
	}
	if !invoice.Status.Match(engine.DRAFT_I) {
		return nil, errors.New("not allowed add transaction (invoice is not draft)")
	}

	strategy := strings.TrimSpace(strings.ToLower(req.GetStrategy()))

	name := strategies.ExistTrName(strategy)
	if name == strategies.UNDEFINED_TR {
		return nil, api.MakeError(codes.NotFound, "Strategy is not found.")
	}
	str := strategies.GetTransactionStrategy(name)
	if str != nil && str.MetaValidation(req.GetMeta()) != nil {
		return nil, api.MakeError(codes.InvalidArgument, "Meta is not validate.")
	}

	var txID int64
	err := s.db.InTransaction(func(tx *reform.TX) error {
		tr := &engine.Transaction{
			ClientID:  &clientID,
			InvoiceID: req.GetInvoiceId(),
			Strategy:  strategy,
			Amount:    req.GetAmount(),
			Key:       req.GetKey(),
			Meta:      req.GetMeta(),
			Status:    engine.DRAFT_TX,
			Provider:  str.Provider(),
		}
		if req.GetKey() != nil {
			key := strings.TrimSpace(strings.ToLower(*req.GetKey()))
			tr.Key = &key
		}
		if err := tx.Insert(tr); err != nil {
			return errors.Wrap(err, "failed insert new transaction")
		}
		txID = tr.TransactionID
		for _, op := range req.GetOperations() {
			o := engine.Operation{
				TransactionID: tr.TransactionID,
				InvoiceID:     tr.InvoiceID,
				SrcAccID:      op.GetSrcAccId(),
				DstAccID:      op.GetDstAccId(),
				Hold:          op.GetHold(),
				HoldAccID:     op.GetHoldAccId(),
				Strategy:      mapStrategyOperFromApiStrategyOper[op.GetStrategy()],
				Amount:        op.GetAmount(),
				Key:           op.GetKey(),
				Meta:          op.GetMeta(),
				Status:        engine.DRAFT_OP,
				UpdatedAt:     time.Now(),
				CreatedAt:     time.Now(),
			}
			if err := tx.Insert(&o); err != nil {
				return errors.Wrap(err, "Failed insert new operation")
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &api.AddTransactionToInvoiceResponse{TxId: txID}, nil
}

func (s Server) GetTransactionByIDs(ctx context.Context, req *api.GetTransactionByIDsRequest) (*api.GetTransactionByIDsResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	txIDs := make([]string, 0, len(req.GetTxIds()))
	for _, v := range req.GetTxIds() {
		txIDs = append(txIDs, strconv.FormatInt(v, 10))
	}
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.StringAttribute("tx_ids", strings.Join(txIDs, ", ")),
	)
	list, err := s.db.SelectAllFrom(
		engine.ViewTransactionTable,
		"WHERE client_id = $1 AND tx_id = ANY ($2)",
		clientID,
		pq.Int64Array(req.GetTxIds()),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed get transaction by IDs.")
	}
	transactions := make([]*api.Tx, 0, len(list))
	for _, v := range list {
		tr := v.(*engine.ViewTransaction)
		var nextStatusTr api.TxStatus
		if tr.NextStatus != nil {
			nextStatusTr = MapTrStatusToApiTrStatus[*tr.NextStatus]
		}
		opers := make([]*api.Oper, 0, len(tr.Operations))
		for _, op := range tr.Operations {
			opers = append(opers, &api.Oper{
				OperId:    op.OperationID,
				InvoiceId: op.InvoiceID,
				TxId:      op.TransactionID,
				SrcAccId:  op.SrcAccID,
				Hold:      op.Hold,
				HoldAccId: op.HoldAccID,
				DstAccId:  op.DstAccID,
				Strategy:  MapOperStrategyToApiTrStrategy[op.Strategy],
				Amount:    op.Amount,
				Key:       op.Key,
				Meta:      op.Meta,
				Status:    MapOperStatusToApiTrStatus[op.Status],
				CreatedAt: &op.CreatedAt,
				UpdatedAt: &op.UpdatedAt,
			})
		}
		var meta *[]byte
		if tr.Meta != nil {
			b := []byte(*tr.Meta)
			meta = &b
		}
		transactions = append(transactions, &api.Tx{
			TxId:               tr.TransactionID,
			InvoiceId:          tr.InvoiceID,
			Key:                tr.Key,
			Strategy:           tr.Strategy,
			Amount:             tr.Amount,
			Provider:           MapTrProviderToApiTrProvider[tr.Provider],
			ProviderOperId:     tr.ProviderOperID,
			ProviderOperStatus: tr.ProviderOperStatus,
			ProviderOperUrl:    tr.ProviderOperUrl,
			Meta:               meta,
			Status:             MapTrStatusToApiTrStatus[tr.Status],
			NextStatus:         nextStatusTr,
			CreatedAt:          &tr.CreatedAt,
			UpdatedAt:          &tr.UpdatedAt,
			Operations:         opers,
		})
	}
	return &api.GetTransactionByIDsResponse{
		Transactions: transactions,
	}, nil
}

func (s Server) AuthInvoice(ctx context.Context, req *api.AuthInvoiceRequest) (*api.AuthInvoiceResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.Int64Attribute("invoice_id", req.GetInvoiceId()),
	)
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	if invoice.ClientID == nil || *invoice.ClientID != clientID {
		return nil, api.MakeError(codes.NotFound, "Invoice is not found.")
	}
	if !invoice.Status.Match(engine.DRAFT_I) {
		return nil, errors.New("not transition to auth (invoice is not draft)")
	}
	if _, err := s.fs.Collection("messages").NewDoc().Create(context.Background(), struct {
		Type      string `firestore:"type"`
		StatusMsg string `firestore:"status_msg"`
		CreatedAt int64  `firestore:"created_at"`
		strategies.MessageUpdateInvoice
	}{
		Type:      strategies.UPDATE_INVOICE_SUBJECT,
		StatusMsg: "new",
		CreatedAt: time.Now().UnixNano(),
		MessageUpdateInvoice: strategies.MessageUpdateInvoice{
			ClientID:  invoice.ClientID,
			InvoiceID: invoice.InvoiceID,
			Strategy:  invoice.Strategy,
			Status:    engine.AUTH_I,
		},
	}); err != nil {
		return nil, errors.Wrap(err, "Failed create message for update status invoice")
	}
	return &api.AuthInvoiceResponse{}, nil
}

func (s Server) AcceptInvoice(ctx context.Context, req *api.AcceptInvoiceRequest) (*api.AcceptInvoiceResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.Int64Attribute("invoice_id", req.GetInvoiceId()),
	)
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	if invoice.ClientID == nil || *invoice.ClientID != clientID {
		return nil, api.MakeError(codes.NotFound, "Invoice is not found.")
	}
	if _, err := s.fs.Collection("messages").NewDoc().Create(context.Background(), struct {
		Type      string `firestore:"type"`
		StatusMsg string `firestore:"status_msg"`
		CreatedAt int64  `firestore:"created_at"`
		strategies.MessageUpdateInvoice
	}{
		Type:      strategies.UPDATE_INVOICE_SUBJECT,
		StatusMsg: "new",
		CreatedAt: time.Now().UnixNano(),
		MessageUpdateInvoice: strategies.MessageUpdateInvoice{
			ClientID:  invoice.ClientID,
			InvoiceID: invoice.InvoiceID,
			Strategy:  invoice.Strategy,
			Status:    engine.MACCEPTED_I,
		},
	}); err != nil {
		return nil, errors.Wrap(err, "Failed create message for update status invoice")
	}
	return &api.AcceptInvoiceResponse{}, nil
}

func (s Server) RejectInvoice(ctx context.Context, req *api.RejectInvoiceRequest) (*api.RejectInvoiceResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.Int64Attribute("invoice_id", req.GetInvoiceId()),
	)
	invoice := &engine.Invoice{InvoiceID: req.GetInvoiceId()}
	if err := s.db.Reload(invoice); err != nil {
		return nil, errors.Wrap(err, "failed find invocie by ID")
	}
	if invoice.ClientID == nil || *invoice.ClientID != clientID {
		return nil, api.MakeError(codes.NotFound, "Invoice is not found.")
	}
	if _, err := s.fs.Collection("messages").NewDoc().Create(context.Background(), struct {
		Type      string `firestore:"type"`
		StatusMsg string `firestore:"status_msg"`
		CreatedAt int64  `firestore:"created_at"`
		strategies.MessageUpdateInvoice
	}{
		Type:      strategies.UPDATE_INVOICE_SUBJECT,
		StatusMsg: "new",
		CreatedAt: time.Now().UnixNano(),
		MessageUpdateInvoice: strategies.MessageUpdateInvoice{
			ClientID:  invoice.ClientID,
			InvoiceID: invoice.InvoiceID,
			Strategy:  invoice.Strategy,
			Status:    engine.MREJECTED_I,
		},
	}); err != nil {
		return nil, errors.Wrap(err, "Failed create message for update status invoice")
	}
	return &api.RejectInvoiceResponse{}, nil
}

func (s Server) AuthTx(ctx context.Context, req *api.AuthTxRequest) (*api.AuthTxResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.Int64Attribute("tx_id", req.GetTxId()),
	)
	tx := &engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(tx); err != nil {
		return nil, errors.Wrap(err, "failed find transaction by ID")
	}
	if tx.ClientID == nil || *tx.ClientID != clientID {
		return nil, api.MakeError(codes.NotFound, "Transaction is not found.")
	}
	if !tx.Status.Match(engine.DRAFT_TX) {
		return nil, errors.New("not transition to auth (transaction is not draft)")
	}
	if _, err := s.fs.Collection("messages").NewDoc().Create(context.Background(), struct {
		Type      string `firestore:"type"`
		StatusMsg string `firestore:"status_msg"`
		CreatedAt int64  `firestore:"created_at"`
		strategies.MessageUpdateTransaction
	}{
		Type:      strategies.UPDATE_TRANSACTION_SUBJECT,
		StatusMsg: "new",
		CreatedAt: time.Now().UnixNano(),
		MessageUpdateTransaction: strategies.MessageUpdateTransaction{
			ClientID:      tx.ClientID,
			TransactionID: tx.TransactionID,
			Strategy:      tx.Strategy,
			Status:        engine.AUTH_TX,
		},
	}); err != nil {
		return nil, errors.Wrap(err, "Failed create message for update status transaction")
	}
	return &api.AuthTxResponse{}, nil
}

func (s Server) AcceptTx(ctx context.Context, req *api.AcceptTxRequest) (*api.AcceptTxResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.Int64Attribute("tx_id", req.GetTxId()),
	)
	tx := &engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(tx); err != nil {
		return nil, errors.Wrap(err, "failed find transaction by ID")
	}
	if tx.ClientID == nil || *tx.ClientID != clientID {
		return nil, api.MakeError(codes.NotFound, "Transaction is not found.")
	}
	if _, err := s.fs.Collection("messages").NewDoc().Create(context.Background(), struct {
		Type      string `firestore:"type"`
		StatusMsg string `firestore:"status_msg"`
		CreatedAt int64  `firestore:"created_at"`
		strategies.MessageUpdateTransaction
	}{
		Type:      strategies.UPDATE_TRANSACTION_SUBJECT,
		StatusMsg: "new",
		CreatedAt: time.Now().UnixNano(),
		MessageUpdateTransaction: strategies.MessageUpdateTransaction{
			ClientID:      tx.ClientID,
			TransactionID: tx.TransactionID,
			Strategy:      tx.Strategy,
			Status:        engine.ACCEPTED_TX,
		},
	}); err != nil {
		return nil, errors.Wrap(err, "Failed create message for update status transaction")
	}
	return &api.AcceptTxResponse{}, nil
}

func (s Server) RejectTx(ctx context.Context, req *api.RejectTxRequest) (*api.RejectTxResponse, error) {
	clientID := services.GetClient(ctx).ClientID
	ctx, span := trace.StartSpan(ctx, "ProcessingRequest")
	defer span.End()
	span.AddAttributes(
		trace.Int64Attribute("client_id", clientID),
		trace.Int64Attribute("tx_id", req.GetTxId()),
	)
	tx := &engine.Transaction{TransactionID: req.GetTxId()}
	if err := s.db.Reload(tx); err != nil {
		return nil, errors.Wrap(err, "failed find transaction by ID")
	}
	if tx.ClientID == nil || *tx.ClientID != clientID {
		return nil, api.MakeError(codes.NotFound, "Transaction is not found.")
	}
	if _, err := s.fs.Collection("messages").NewDoc().Create(context.Background(), struct {
		Type      string `firestore:"type"`
		StatusMsg string `firestore:"status_msg"`
		CreatedAt int64  `firestore:"created_at"`
		strategies.MessageUpdateTransaction
	}{
		Type:      strategies.UPDATE_TRANSACTION_SUBJECT,
		StatusMsg: "new",
		CreatedAt: time.Now().UnixNano(),
		MessageUpdateTransaction: strategies.MessageUpdateTransaction{
			ClientID:      tx.ClientID,
			TransactionID: tx.TransactionID,
			Strategy:      tx.Strategy,
			Status:        engine.REJECTED_TX,
		},
	}); err != nil {
		return nil, errors.Wrap(err, "Failed create message for update status transaction")
	}
	return &api.RejectTxResponse{}, nil
}

var MapInvStatusToApiInvStatus = map[engine.InvoiceStatus]api.InvoiceStatus{
	engine.DRAFT_I:     api.InvoiceStatus_DRAFT_I,
	engine.AUTH_I:      api.InvoiceStatus_AUTH_I,
	engine.WAIT_I:      api.InvoiceStatus_WAIT_I,
	engine.ACCEPTED_I:  api.InvoiceStatus_ACCEPTED_I,
	engine.MACCEPTED_I: api.InvoiceStatus_MACCEPTED_I,
	engine.REJECTED_I:  api.InvoiceStatus_REJECTED_I,
	engine.MREJECTED_I: api.InvoiceStatus_MREJECTED_I,
}

var mapStrategyOperFromApiStrategyOper = map[api.OperStrategy]engine.OperationStrategy{
	api.OperStrategy_SIMPLE_OPS:   engine.SIMPLE_OPS,
	api.OperStrategy_RECHARGE_OPS: engine.RECHARGE_OPS,
	api.OperStrategy_WITHDRAW_OPS: engine.WITHDRAW_OPS,
}

var MapOperStrategyToApiTrStrategy = map[engine.OperationStrategy]api.OperStrategy{
	engine.SIMPLE_OPS:   api.OperStrategy_SIMPLE_OPS,
	engine.RECHARGE_OPS: api.OperStrategy_RECHARGE_OPS,
	engine.WITHDRAW_OPS: api.OperStrategy_WITHDRAW_OPS,
}

var MapTrProviderToApiTrProvider = map[provider.Provider]api.Provider{
	provider.INTERNAL: api.Provider_INTERNAL_PROVIDER,
	sberbank.SBERBANK: api.Provider_SBERBANK_PROVIDER,
}

var MapTrStatusToApiTrStatus = map[engine.TransactionStatus]api.TxStatus{
	engine.DRAFT_TX:     api.TxStatus_DRAFT_TX,
	engine.AUTH_TX:      api.TxStatus_AUTH_TX,
	engine.WAUTH_TX:     api.TxStatus_AUTH_TX,
	engine.HOLD_TX:      api.TxStatus_HOLD_TX,
	engine.WHOLD_TX:     api.TxStatus_WHOLD_TX,
	engine.ACCEPTED_TX:  api.TxStatus_ACCEPTED_TX,
	engine.WACCEPTED_TX: api.TxStatus_WACCEPTED_TX,
	engine.REJECTED_TX:  api.TxStatus_REJECTED_TX,
	engine.WREJECTED_TX: api.TxStatus_WREJECTED_TX,
	engine.FAILED_TX:    api.TxStatus_FAILED_TX,
}

var MapOperStatusToApiTrStatus = map[engine.OperationStatus]api.OperStatus{
	engine.DRAFT_OP:    api.OperStatus_DRAFT_OP,
	engine.HOLD_OP:     api.OperStatus_HOLD_OP,
	engine.ACCEPTED_OP: api.OperStatus_ACCEPTED_OP,
	engine.REJECTED_OP: api.OperStatus_REJECTED_OP,
}
