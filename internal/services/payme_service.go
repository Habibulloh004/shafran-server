package services

import (
	"context"
	"errors"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/example/shafran/internal/models"
)

// TransactionState mirrors the JS TransactionState enum.
const (
	TransactionStatePaid           = 2
	TransactionStatePending        = 1
	TransactionStatePendingCanceled = -1
	TransactionStatePaidCanceled   = -2
)

// PaymeErrorInfo describes a Payme-compatible error.
type PaymeErrorInfo struct {
	Name    string
	Code    int
	Message map[string]string
}

var (
	PaymeErrorInvalidAmount = PaymeErrorInfo{
		Name: "InvalidAmount",
		Code: -31001,
		Message: map[string]string{
			"uz": "Noto'g'ri summa",
			"ru": "Недопустимая сумма",
			"en": "Invalid amount",
		},
	}
	PaymeErrorCantDoOperation = PaymeErrorInfo{
		Name: "CantDoOperation",
		Code: -31008,
		Message: map[string]string{
			"uz": "Biz operatsiyani bajara olmaymiz",
			"ru": "Мы не можем сделать операцию",
			"en": "We can't do operation",
		},
	}
	PaymeErrorTransactionNotFound = PaymeErrorInfo{
		Name: "TransactionNotFound",
		Code: -31050,
		Message: map[string]string{
			"uz": "Tranzaktsiya topilmadi",
			"ru": "Транзакция не найдена",
			"en": "Transaction not found",
		},
	}
	PaymeErrorAlreadyDone = PaymeErrorInfo{
		Name: "AlreadyDone",
		Code: -31060,
		Message: map[string]string{
			"uz": "Mahsulot uchun to'lov qilingan",
			"ru": "Оплачено за товар",
			"en": "Paid for the product",
		},
	}
	PaymeErrorPending = PaymeErrorInfo{
		Name: "Pending",
		Code: -31050,
		Message: map[string]string{
			"uz": "Mahsulot uchun to'lov kutilayapti",
			"ru": "Ожидается оплата товар",
			"en": "Payment for the product is pending",
		},
	}
	PaymeErrorInvalidAuthorization = PaymeErrorInfo{
		Name: "InvalidAuthorization",
		Code: -32504,
		Message: map[string]string{
			"uz": "Avtorizatsiya yaroqsiz",
			"ru": "Авторизация недействительна",
			"en": "Authorization invalid",
		},
	}
)

// TransactionError is a structured Payme transaction error.
type TransactionError struct {
	Info PaymeErrorInfo
	ID   any
	Data any
}

func (e *TransactionError) Error() string {
	return e.Info.Name
}

// PaymeService ports business logic from the JS payme.service.
type PaymeService struct {
	db *gorm.DB
}

func NewPaymeService(db *gorm.DB) *PaymeService {
	return &PaymeService{db: db}
}

type PaymeAccount struct {
	OrderID string `json:"order_id"`
}

type CheckPerformParams struct {
	Amount  int64       `json:"amount"`
	Account PaymeAccount `json:"account"`
}

type CheckTransactionParams struct {
	ID any `json:"id"`
}

type CreateTransactionParams struct {
	Account PaymeAccount `json:"account"`
	Time    int64        `json:"time"`
	Amount  int64        `json:"amount"`
	ID      string       `json:"id"`
}

type PerformTransactionParams struct {
	ID string `json:"id"`
}

type CancelTransactionParams struct {
	ID     string `json:"id"`
	Reason int    `json:"reason"`
}

type StatementParams struct {
	From int64 `json:"from"`
	To   int64 `json:"to"`
}

type CheckTransactionResult struct {
	CreateTime  int64 `json:"create_time"`
	PerformTime int64 `json:"perform_time"`
	CancelTime  int64 `json:"cancel_time"`
	Transaction string `json:"transaction"`
	State       int    `json:"state"`
	Reason      *int   `json:"reason"`
}

type PerformTransactionResult struct {
	PerformTime int64  `json:"perform_time"`
	Transaction string `json:"transaction"`
	State       int    `json:"state"`
}

type CancelTransactionResult struct {
	CancelTime  int64  `json:"cancel_time"`
	Transaction string `json:"transaction"`
	State       int    `json:"state"`
}

type StatementTransaction struct {
	TransactionID string         `json:"transaction_id"`
	Time          int64          `json:"time"`
	Amount        int64          `json:"amount"`
	Account       PaymeAccount   `json:"account"`
	CreateTime    int64          `json:"create_time"`
	PerformTime   int64          `json:"perform_time"`
	CancelTime    int64          `json:"cancel_time"`
	Transaction   string         `json:"transaction"`
	State         int            `json:"state"`
	Reason        *int           `json:"reason"`
}

// CheckPerformTransaction validates that the order exists and amount matches.
func (s *PaymeService) CheckPerformTransaction(ctx context.Context, params CheckPerformParams, id any) error {
	amount := params.Amount / 100

	var txn models.PaymeTransaction
	if err := s.db.WithContext(ctx).
		Where("id = ? AND provider = ?", params.Account.OrderID, "payme").
		First(&txn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &TransactionError{Info: PaymeErrorTransactionNotFound, ID: id}
		}
		return err
	}

	if txn.Amount != amount {
		return &TransactionError{Info: PaymeErrorInvalidAmount, ID: id}
	}

	return nil
}

// CheckTransaction returns transaction state by transaction id.
func (s *PaymeService) CheckTransaction(ctx context.Context, params CheckTransactionParams, id any) (*CheckTransactionResult, error) {
	var lookupID string
	switch v := params.ID.(type) {
	case string:
		lookupID = v
	case float64:
		lookupID = strconv.FormatInt(int64(v), 10)
	default:
		return nil, &TransactionError{Info: PaymeErrorTransactionNotFound, ID: id}
	}

	var txn models.PaymeTransaction
	if err := s.db.WithContext(ctx).
		Where("transaction_id = ?", lookupID).
		First(&txn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &TransactionError{Info: PaymeErrorTransactionNotFound, ID: id}
		}
		return nil, err
	}

	var reason *int
	if txn.Reason != nil && *txn.Reason != 0 {
		reason = txn.Reason
	}

	return &CheckTransactionResult{
		CreateTime:  txn.CreateTime,
		PerformTime: txn.PerformTime,
		CancelTime:  txn.CancelTime,
		Transaction: txn.TransactionID,
		State:       txn.Status,
		Reason:      reason,
	}, nil
}

// CreateTransaction creates or reuses a pending transaction for the given order.
func (s *PaymeService) CreateTransaction(ctx context.Context, params CreateTransactionParams, id any) (*CheckTransactionResult, error) {
	if err := s.CheckPerformTransaction(ctx, CheckPerformParams{
		Amount:  params.Amount,
		Account: params.Account,
	}, id); err != nil {
		return nil, err
	}

	currentTime := time.Now().UnixMilli()

	var existing models.PaymeTransaction
	err := s.db.WithContext(ctx).
		Where("transaction_id = ?", params.ID).
		First(&existing).Error
	if err == nil {
		if existing.Status != TransactionStatePending {
			return nil, &TransactionError{Info: PaymeErrorCantDoOperation, ID: id}
		}

		if (currentTime-existing.CreateTime)/60000 >= 12 {
			if err := s.db.WithContext(ctx).
				Model(&models.PaymeTransaction{}).
				Where("transaction_id = ?", params.ID).
				Updates(map[string]any{
					"status": TransactionStatePendingCanceled,
					"reason": 4,
				}).Error; err != nil {
				return nil, err
			}
			return nil, &TransactionError{Info: PaymeErrorCantDoOperation, ID: id}
		}

		return &CheckTransactionResult{
			CreateTime:  existing.CreateTime,
			Transaction: params.ID,
			State:       TransactionStatePending,
		}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var order models.PaymeTransaction
	if err := s.db.WithContext(ctx).
		Where("id = ? AND provider = ?", params.Account.OrderID, "payme").
		First(&order).Error; err == nil {
		if order.Status == TransactionStatePaid {
			return nil, &TransactionError{Info: PaymeErrorAlreadyDone, ID: id}
		}
		if order.Status == TransactionStatePending {
			return nil, &TransactionError{Info: PaymeErrorPending, ID: id}
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if err := s.db.WithContext(ctx).
		Model(&models.PaymeTransaction{}).
		Where("id = ?", params.Account.OrderID).
		Updates(map[string]any{
			"transaction_id": params.ID,
			"status":         TransactionStatePending,
			"create_time":    params.Time,
		}).Error; err != nil {
		return nil, err
	}

	return &CheckTransactionResult{
		Transaction: params.ID,
		State:       TransactionStatePending,
		CreateTime:  params.Time,
	}, nil
}

// PerformTransaction marks a pending transaction as paid.
// Note: external side effects (Poster, Abdugani, Telegram) are not replicated here.
func (s *PaymeService) PerformTransaction(ctx context.Context, params PerformTransactionParams, id any) (*PerformTransactionResult, error) {
	currentTime := time.Now().UnixMilli()

	var txn models.PaymeTransaction
	if err := s.db.WithContext(ctx).
		Where("transaction_id = ?", params.ID).
		First(&txn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &TransactionError{Info: PaymeErrorTransactionNotFound, ID: id}
		}
		return nil, err
	}

	if txn.Status != TransactionStatePending {
		if txn.Status != TransactionStatePaid {
			return nil, &TransactionError{Info: PaymeErrorCantDoOperation, ID: id}
		}
		return &PerformTransactionResult{
			PerformTime: txn.PerformTime,
			Transaction: txn.TransactionID,
			State:       TransactionStatePaid,
		}, nil
	}

	if (currentTime-txn.CreateTime)/60000 >= 12 {
		if err := s.db.WithContext(ctx).
			Model(&models.PaymeTransaction{}).
			Where("transaction_id = ?", params.ID).
			Updates(map[string]any{
				"status":      TransactionStatePendingCanceled,
				"reason":      4,
				"cancel_time": currentTime,
			}).Error; err != nil {
			return nil, err
		}
		return nil, &TransactionError{Info: PaymeErrorCantDoOperation, ID: id}
	}

	if err := s.db.WithContext(ctx).
		Model(&models.PaymeTransaction{}).
		Where("transaction_id = ?", params.ID).
		Updates(map[string]any{
			"status":       TransactionStatePaid,
			"perform_time": currentTime,
		}).Error; err != nil {
		return nil, err
	}

	return &PerformTransactionResult{
		PerformTime: currentTime,
		Transaction: txn.TransactionID,
		State:       TransactionStatePaid,
	}, nil
}

// CancelTransaction cancels an existing transaction.
func (s *PaymeService) CancelTransaction(ctx context.Context, params CancelTransactionParams, id any) (*CancelTransactionResult, error) {
	var txn models.PaymeTransaction
	if err := s.db.WithContext(ctx).
		Where("transaction_id = ?", params.ID).
		First(&txn).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &TransactionError{Info: PaymeErrorTransactionNotFound, ID: id}
		}
		return nil, err
	}

	currentTime := time.Now().UnixMilli()

	if txn.Status > 0 {
		newState := -1 * intAbs(txn.Status)
		if err := s.db.WithContext(ctx).
			Model(&models.PaymeTransaction{}).
			Where("transaction_id = ?", params.ID).
			Updates(map[string]any{
				"status":      newState,
				"reason":      params.Reason,
				"cancel_time": currentTime,
			}).Error; err != nil {
			return nil, err
		}
		txn.Status = newState
		txn.CancelTime = currentTime
	}

	cancelTime := txn.CancelTime
	if cancelTime == 0 {
		cancelTime = currentTime
	}

	return &CancelTransactionResult{
		CancelTime:  cancelTime,
		Transaction: txn.TransactionID,
		State:       -1 * intAbs(txn.Status),
	}, nil
}

// GetStatement returns transactions in the given time range.
func (s *PaymeService) GetStatement(ctx context.Context, params StatementParams) ([]StatementTransaction, error) {
	var txns []models.PaymeTransaction
	if err := s.db.WithContext(ctx).
		Where("create_time >= ? AND create_time <= ? AND provider = ?", params.From, params.To, "payme").
		Find(&txns).Error; err != nil {
		return nil, err
	}

	result := make([]StatementTransaction, 0, len(txns))
	for _, t := range txns {
		result = append(result, StatementTransaction{
			TransactionID: t.TransactionID,
			Time:          t.CreateTime,
			Amount:        t.Amount,
			Account:       PaymeAccount{OrderID: t.ID.String()},
			CreateTime:    t.CreateTime,
			PerformTime:   t.PerformTime,
			CancelTime:    t.CancelTime,
			Transaction:   t.TransactionID,
			State:         t.Status,
			Reason:        t.Reason,
		})
	}

	return result, nil
}

func intAbs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
