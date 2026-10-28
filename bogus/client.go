package bogus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Flying-Tea-Squad/paykit-go"
)

var (
	// ErrNilRequest is returned when a nil request pointer is provided to any gateway method.
	ErrNilRequest = errors.New("bogus: request cannot be nil")

	// ErrUserCancelled indicates the simulated customer cancelled the payment prompt.
	ErrUserCancelled = errors.New("bogus: user cancelled transaction")

	// ErrPinTimeout indicates the simulated customer did not enter their PIN before timeout.
	ErrPinTimeout = errors.New("bogus: pin entry timeout")

	// ErrInsufficientFunds indicates simulated insufficient funds for disbursement.
	ErrInsufficientFunds = paykit.ErrInsufficientFundsSentinel
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

// NewClient is an alias for New.
func NewClient() *Client {
	return New()
}

// Name returns the canonical provider name for the test double.
func (c *Client) Name() string {
	return "bogus"
}

// Charge deterministically simulates payment collection based on req.Amount.Amount:
//   - Ends in 00 -> StatusSuccess
//   - Ends in 05 -> StatusFailed with ErrUserCancelled
//   - Ends in 02 -> StatusFailed with ErrPinTimeout
//   - Otherwise  -> StatusProcessing
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

	amount := req.Amount.Amount()
	if amount < 0 {
		amount = -amount
	}
	suffix := amount % 100

	txnRef := req.Reference
	if txnRef == "" {
		txnRef = fmt.Sprintf("txn_%d", req.Amount.Amount())
	}

	switch suffix {
	case 0:
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
	case 5:
		return &paykit.ChargeResponse{
			Success:       false,
			Message:       ErrUserCancelled.Error(),
			TransactionID: "fail_chg_" + txnRef,
			Status:        paykit.StatusFailed,
			Raw:           json.RawMessage(`{"status":"failed","error":"user_cancelled","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, ErrUserCancelled
	case 2:
		return &paykit.ChargeResponse{
			Success:       false,
			Message:       ErrPinTimeout.Error(),
			TransactionID: "fail_chg_" + txnRef,
			Status:        paykit.StatusFailed,
			Raw:           json.RawMessage(`{"status":"failed","error":"pin_timeout","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, ErrPinTimeout
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

// Disburse deterministically simulates funds disbursement based on req.Amount.Amount:
//   - Ends in 00 -> StatusSuccess
//   - Ends in 50 -> StatusFailed with ErrInsufficientFunds
//   - Otherwise  -> StatusProcessing
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

	amount := req.Amount.Amount()
	if amount < 0 {
		amount = -amount
	}
	suffix := amount % 100

	txnRef := req.Reference
	if txnRef == "" {
		txnRef = fmt.Sprintf("disb_%d", req.Amount.Amount())
	}

	switch suffix {
	case 0:
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
	case 50:
		return &paykit.DisbursementResponse{
			Success:       false,
			Message:       ErrInsufficientFunds.Error(),
			TransactionID: "fail_disb_" + txnRef,
			Status:        paykit.StatusFailed,
			Raw:           json.RawMessage(`{"status":"failed","error":"insufficient_funds","provider":"bogus"}`),
			Metadata: map[string]any{
				"provider": "bogus",
				"phone":    req.Phone,
			},
		}, ErrInsufficientFunds
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
