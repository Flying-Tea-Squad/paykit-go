package bogus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Flying-Tea-Squad/paykit-go"
)

const (
	// SuffixSuccess routes Charge and Disburse operations to StatusSuccess.
	SuffixSuccess = 0

	// SuffixPinTimeout routes Charge operations to StatusTimeout simulating customer PIN entry timeout.
	SuffixPinTimeout = 2

	// SuffixUserCancelled routes Charge operations to StatusCancelled simulating customer cancellation.
	SuffixUserCancelled = 5

	// SuffixInsufficientFunds routes Disburse operations to StatusFailed simulating insufficient account balance.
	SuffixInsufficientFunds = 50
)

var (
	// ErrNilRequest is returned when a nil request pointer is provided to any gateway method.
	ErrNilRequest = errors.New("bogus: request cannot be nil")

	// ErrChargeAmountInvalid is returned when a charge amount is not greater than zero.
	ErrChargeAmountInvalid = errors.New("bogus: charge amount must be greater than zero")

	// ErrDisburseAmountInvalid is returned when a disbursement amount is not greater than zero.
	ErrDisburseAmountInvalid = errors.New("bogus: disburse amount must be greater than zero")

	// ErrTransactionIDRequired is returned when a transaction ID is missing or empty.
	ErrTransactionIDRequired = errors.New("bogus: transaction ID is required")
)

// Client is a zero-dependency in-memory test double gateway implementing paykit.Gateway
// and paykit.Disburser. It routes operations deterministically based on request amounts
// and identifiers.
type Client struct{}

// Compile-time interface satisfaction guarantees.
var (
	_ paykit.Gateway   = (*Client)(nil)
	_ paykit.Disburser = (*Client)(nil)
)

func init() {
	paykit.Register("bogus", func() (paykit.Gateway, error) {
		return New(), nil
	})
}

// New returns a new initialized bogus Client.
func New() *Client {
	return &Client{}
}

// Name returns the canonical provider name for the test double.
func (c *Client) Name() string {
	return "bogus"
}

// Charge deterministically simulates payment collection based on req.Amount.Amount() % 100:
//   - SuffixSuccess (00)       -> StatusSuccess
//   - SuffixUserCancelled (05) -> StatusCancelled
//   - SuffixPinTimeout (02)    -> StatusTimeout
//   - Otherwise                -> StatusProcessing
func (c *Client) Charge(ctx context.Context, req *paykit.ChargeRequest) (*paykit.ChargeResponse, error) {
	if ctx == nil {
		return nil, errors.New("bogus: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, ErrNilRequest
	}

	if req.Amount.Amount() <= 0 {
		return nil, ErrChargeAmountInvalid
	}
	suffix := req.Amount.Amount() % 100

	txnRef := req.Reference
	if txnRef == "" {
		txnRef = fmt.Sprintf("txn_%d", req.Amount.Amount())
	}

	switch suffix {
	case SuffixSuccess:
		return &paykit.ChargeResponse{
			Success:       true,
			Message:       "charge successful",
			TransactionID: "bogus_chg_" + txnRef,
			Status:        paykit.StatusSuccess,
			Raw:           json.RawMessage(`{"status":"success","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, nil
	case SuffixUserCancelled:
		return &paykit.ChargeResponse{
			Success:       false,
			Message:       "user cancelled transaction",
			TransactionID: "fail_chg_" + txnRef,
			Status:        paykit.StatusCancelled,
			Raw:           json.RawMessage(`{"status":"cancelled","error":"user_cancelled","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, nil
	case SuffixPinTimeout:
		return &paykit.ChargeResponse{
			Success:       false,
			Message:       "pin entry timeout",
			TransactionID: "fail_chg_" + txnRef,
			Status:        paykit.StatusTimeout,
			Raw:           json.RawMessage(`{"status":"timeout","error":"pin_timeout","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, nil
	default:
		return &paykit.ChargeResponse{
			Success:       true,
			Message:       "charge processing",
			TransactionID: "bogus_chg_" + txnRef,
			Status:        paykit.StatusProcessing,
			Raw:           json.RawMessage(`{"status":"processing","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, nil
	}
}

// Disburse deterministically simulates funds disbursement based on req.Amount.Amount() % 100:
//   - SuffixSuccess (00)           -> StatusSuccess
//   - SuffixInsufficientFunds (50) -> StatusFailed
//   - Otherwise                    -> StatusProcessing
func (c *Client) Disburse(ctx context.Context, req *paykit.DisbursementRequest) (*paykit.DisbursementResponse, error) {
	if ctx == nil {
		return nil, errors.New("bogus: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, ErrNilRequest
	}

	if req.Amount.Amount() <= 0 {
		return nil, ErrDisburseAmountInvalid
	}
	suffix := req.Amount.Amount() % 100

	txnRef := req.Reference
	if txnRef == "" {
		txnRef = fmt.Sprintf("txn_%d", req.Amount.Amount())
	}

	switch suffix {
	case SuffixSuccess:
		return &paykit.DisbursementResponse{
			Success:       true,
			Message:       "disbursement successful",
			TransactionID: "bogus_disb_" + txnRef,
			Status:        paykit.StatusSuccess,
			Raw:           json.RawMessage(`{"status":"success","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, nil
	case SuffixInsufficientFunds:
		return &paykit.DisbursementResponse{
			Success:       false,
			Message:       "insufficient funds",
			TransactionID: "fail_disb_" + txnRef,
			Status:        paykit.StatusFailed,
			Raw:           json.RawMessage(`{"status":"failed","error":"insufficient_funds","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, nil
	default:
		return &paykit.DisbursementResponse{
			Success:       true,
			Message:       "disbursement processing",
			TransactionID: "bogus_disb_" + txnRef,
			Status:        paykit.StatusProcessing,
			Raw:           json.RawMessage(`{"status":"processing","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, nil
	}
}

// QueryStatus deterministically simulates transaction status inquiries:
//   - TransactionID prefixed with "fail" -> StatusFailed
//   - Any other ID                       -> StatusSuccess
func (c *Client) QueryStatus(ctx context.Context, req *paykit.StatusRequest) (*paykit.StatusResponse, error) {
	if ctx == nil {
		return nil, errors.New("bogus: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, ErrNilRequest
	}

	if strings.TrimSpace(req.TransactionID) == "" {
		return nil, ErrTransactionIDRequired
	}

	if strings.HasPrefix(req.TransactionID, "fail") {
		return &paykit.StatusResponse{
			Success:       false,
			Message:       "transaction failed",
			TransactionID: req.TransactionID,
			Status:        paykit.StatusFailed,
			Raw:           json.RawMessage(`{"status":"failed","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
			},
		}, nil
	}

	return &paykit.StatusResponse{
		Success:       true,
		Message:       "transaction successful",
		TransactionID: req.TransactionID,
		Status:        paykit.StatusSuccess,
		Raw:           json.RawMessage(`{"status":"success","provider":"bogus"}`),
		Metadata: map[string]any{
			"provider": "bogus",
		},
	}, nil
}

// Refund deterministically simulates transaction refunding / reversal:
//   - TransactionID prefixed with "fail" -> StatusFailed
//   - Any other ID                       -> StatusSuccess
func (c *Client) Refund(ctx context.Context, req *paykit.RefundRequest) (*paykit.RefundResponse, error) {
	if ctx == nil {
		return nil, errors.New("bogus: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, ErrNilRequest
	}

	if strings.TrimSpace(req.TransactionID) == "" {
		return nil, ErrTransactionIDRequired
	}

	if strings.HasPrefix(req.TransactionID, "fail") {
		return &paykit.RefundResponse{
			Success:       false,
			Message:       "refund failed",
			RefundID:      "fail_ref_" + req.TransactionID,
			TransactionID: req.TransactionID,
			Status:        paykit.StatusFailed,
			Raw:           json.RawMessage(`{"status":"failed","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
			},
		}, nil
	}

	return &paykit.RefundResponse{
		Success:       true,
		Message:       "refund successful",
		RefundID:      "bogus_ref_" + req.TransactionID,
		TransactionID: req.TransactionID,
		Status:        paykit.StatusSuccess,
		Raw:           json.RawMessage(`{"status":"success","provider":"bogus"}`),
		Metadata: map[string]any{
			"provider": "bogus",
		},
	}, nil
}
