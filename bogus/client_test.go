package bogus

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Flying-Tea-Squad/paykit-go"
)

func TestClient_InterfaceAssertions(t *testing.T) {
	t.Parallel()

	client := New()
	var _ paykit.Gateway = client
	var _ paykit.Disburser = client

	client2 := NewClient()
	if client2 == nil {
		t.Fatal("expected NewClient() to return non-nil Client")
	}

	if client.Name() != "bogus" {
		t.Fatalf("expected Name() to be 'bogus', got %q", client.Name())
	}
}

func TestClient_Registry(t *testing.T) {
	// The bogus client registers itself via init().
	gw, err := paykit.Get("bogus")
	if err != nil {
		t.Fatalf("expected paykit.Get('bogus') to succeed, got: %v", err)
	}

	if gw.Name() != "bogus" {
		t.Fatalf("expected registered gateway name to be 'bogus', got %q", gw.Name())
	}

	disburser, ok := gw.(paykit.Disburser)
	if !ok || disburser == nil {
		t.Fatalf("expected registered gateway to satisfy paykit.Disburser")
	}
}

func TestClient_Charge_TableDriven(t *testing.T) {
	t.Parallel()

	client := New()

	tests := []struct {
		name           string
		req            *paykit.ChargeRequest
		ctx            func() (context.Context, context.CancelFunc)
		wantStatus     paykit.TransactionStatus
		wantSuccess    bool
		wantErr        error
		wantTxnIDPfx   string
	}{
		{
			name:         "amount ends in 00 -> StatusSuccess",
			req:          chargeReqSuccess(),
			wantStatus:   paykit.StatusSuccess,
			wantSuccess:  true,
			wantErr:      nil,
			wantTxnIDPfx: "bogus_chg_",
		},
		{
			name:         "amount ends in 05 -> StatusFailed with ErrUserCancelled",
			req:          chargeReqCancelled(),
			wantStatus:   paykit.StatusFailed,
			wantSuccess:  false,
			wantErr:      ErrUserCancelled,
			wantTxnIDPfx: "fail_chg_",
		},
		{
			name:         "amount ends in 02 -> StatusFailed with ErrPinTimeout",
			req:          chargeReqPinTimeout(),
			wantStatus:   paykit.StatusFailed,
			wantSuccess:  false,
			wantErr:      ErrPinTimeout,
			wantTxnIDPfx: "fail_chg_",
		},
		{
			name:         "amount ends in 99 -> StatusProcessing",
			req:          chargeReqProcessing(),
			wantStatus:   paykit.StatusProcessing,
			wantSuccess:  true,
			wantErr:      nil,
			wantTxnIDPfx: "bogus_chg_",
		},
		{
			name:         "amount ends in 23 -> StatusProcessing",
			req:          buildChargeRequest(123, "+254700000023", "P23"),
			wantStatus:   paykit.StatusProcessing,
			wantSuccess:  true,
			wantErr:      nil,
			wantTxnIDPfx: "bogus_chg_",
		},
		{
			name:         "amount without reference generates fallback txn id",
			req:          &paykit.ChargeRequest{Amount: paykit.NewMoney(500, "KES"), Phone: "+254711111111"},
			wantStatus:   paykit.StatusSuccess,
			wantSuccess:  true,
			wantErr:      nil,
			wantTxnIDPfx: "bogus_chg_txn_500",
		},
		{
			name:       "nil request returns ErrNilRequest",
			req:        nil,
			wantErr:    ErrNilRequest,
		},
		{
			name: "cancelled context returns context.Canceled",
			req:  chargeReqSuccess(),
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
		{
			name: "expired context returns context.DeadlineExceeded",
			req:  chargeReqSuccess(),
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				return ctx, cancel
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.ctx()
				defer cancel()
			}

			resp, err := client.Charge(ctx, tt.req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if tt.req == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					if resp != nil {
						t.Fatalf("expected nil response on guard failure, got %+v", resp)
					}
					return
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("expected non-nil response")
			}

			if resp.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, resp.Status)
			}
			if resp.Success != tt.wantSuccess {
				t.Errorf("expected success %v, got %v", tt.wantSuccess, resp.Success)
			}
			if tt.wantTxnIDPfx != "" && !strings.HasPrefix(resp.TransactionID, tt.wantTxnIDPfx) {
				t.Errorf("expected transaction ID prefix %q, got %q", tt.wantTxnIDPfx, resp.TransactionID)
			}
		})
	}
}

func TestClient_Disburse_TableDriven(t *testing.T) {
	t.Parallel()

	client := New()

	tests := []struct {
		name           string
		req            *paykit.DisbursementRequest
		ctx            func() (context.Context, context.CancelFunc)
		wantStatus     paykit.TransactionStatus
		wantSuccess    bool
		wantErr        error
		wantTxnIDPfx   string
	}{
		{
			name:         "amount ends in 00 -> StatusSuccess",
			req:          disburseReqSuccess(),
			wantStatus:   paykit.StatusSuccess,
			wantSuccess:  true,
			wantErr:      nil,
			wantTxnIDPfx: "bogus_disb_",
		},
		{
			name:         "amount ends in 50 -> StatusFailed with ErrInsufficientFunds",
			req:          disburseReqInsufficientFunds(),
			wantStatus:   paykit.StatusFailed,
			wantSuccess:  false,
			wantErr:      ErrInsufficientFunds,
			wantTxnIDPfx: "fail_disb_",
		},
		{
			name:         "amount ends in 25 -> StatusProcessing",
			req:          disburseReqProcessing(),
			wantStatus:   paykit.StatusProcessing,
			wantSuccess:  true,
			wantErr:      nil,
			wantTxnIDPfx: "bogus_disb_",
		},
		{
			name:         "amount without reference generates fallback txn id",
			req:          &paykit.DisbursementRequest{Amount: paykit.NewMoney(1000, "KES"), Phone: "+254722222222"},
			wantStatus:   paykit.StatusSuccess,
			wantSuccess:  true,
			wantErr:      nil,
			wantTxnIDPfx: "bogus_disb_disb_1000",
		},
		{
			name:       "nil request returns ErrNilRequest",
			req:        nil,
			wantErr:    ErrNilRequest,
		},
		{
			name: "cancelled context returns context.Canceled",
			req:  disburseReqSuccess(),
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
		{
			name: "expired context returns context.DeadlineExceeded",
			req:  disburseReqSuccess(),
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				return ctx, cancel
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.ctx()
				defer cancel()
			}

			resp, err := client.Disburse(ctx, tt.req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if tt.req == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					if resp != nil {
						t.Fatalf("expected nil response on guard failure, got %+v", resp)
					}
					return
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp == nil {
				t.Fatal("expected non-nil response")
			}

			if resp.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, resp.Status)
			}
			if resp.Success != tt.wantSuccess {
				t.Errorf("expected success %v, got %v", tt.wantSuccess, resp.Success)
			}
			if tt.wantTxnIDPfx != "" && !strings.HasPrefix(resp.TransactionID, tt.wantTxnIDPfx) {
				t.Errorf("expected transaction ID prefix %q, got %q", tt.wantTxnIDPfx, resp.TransactionID)
			}
		})
	}
}

func TestClient_QueryStatus_TableDriven(t *testing.T) {
	t.Parallel()

	client := New()

	tests := []struct {
		name        string
		req         *paykit.StatusRequest
		ctx         func() (context.Context, context.CancelFunc)
		wantStatus  paykit.TransactionStatus
		wantSuccess bool
		wantErr     error
	}{
		{
			name:        "TransactionID prefixed with fail -> StatusFailed",
			req:         buildStatusRequest("fail_12345"),
			wantStatus:  paykit.StatusFailed,
			wantSuccess: false,
			wantErr:     nil,
		},
		{
			name:        "TransactionID exactly fail -> StatusFailed",
			req:         buildStatusRequest("fail"),
			wantStatus:  paykit.StatusFailed,
			wantSuccess: false,
			wantErr:     nil,
		},
		{
			name:        "TransactionID with standard ID -> StatusSuccess",
			req:         buildStatusRequest("bogus_chg_999"),
			wantStatus:  paykit.StatusSuccess,
			wantSuccess: true,
			wantErr:     nil,
		},
		{
			name:        "TransactionID arbitrary string -> StatusSuccess",
			req:         buildStatusRequest("random_success_id"),
			wantStatus:  paykit.StatusSuccess,
			wantSuccess: true,
			wantErr:     nil,
		},
		{
			name:    "nil request returns ErrNilRequest",
			req:     nil,
			wantErr: ErrNilRequest,
		},
		{
			name: "cancelled context returns context.Canceled",
			req:  buildStatusRequest("bogus_1"),
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.ctx()
				defer cancel()
			}

			resp, err := client.QueryStatus(ctx, tt.req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if resp != nil {
					t.Fatalf("expected nil response on error, got %+v", resp)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("expected non-nil response")
			}
			if resp.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, resp.Status)
			}
			if resp.Success != tt.wantSuccess {
				t.Errorf("expected success %v, got %v", tt.wantSuccess, resp.Success)
			}
			if resp.TransactionID != tt.req.TransactionID {
				t.Errorf("expected transaction ID %q, got %q", tt.req.TransactionID, resp.TransactionID)
			}
		})
	}
}

func TestClient_Refund_TableDriven(t *testing.T) {
	t.Parallel()

	client := New()

	tests := []struct {
		name        string
		req         *paykit.RefundRequest
		ctx         func() (context.Context, context.CancelFunc)
		wantStatus  paykit.TransactionStatus
		wantSuccess bool
		wantErr     error
	}{
		{
			name:        "standard transaction refund succeeds",
			req:         buildRefundRequest("txn_100", 1000),
			wantStatus:  paykit.StatusSuccess,
			wantSuccess: true,
			wantErr:     nil,
		},
		{
			name:        "TransactionID prefixed with fail returns StatusFailed",
			req:         buildRefundRequest("fail_txn_100", 1000),
			wantStatus:  paykit.StatusFailed,
			wantSuccess: false,
			wantErr:     nil,
		},
		{
			name:    "nil request returns ErrNilRequest",
			req:     nil,
			wantErr: ErrNilRequest,
		},
		{
			name: "cancelled context returns context.Canceled",
			req:  buildRefundRequest("txn_100", 1000),
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			if tt.ctx != nil {
				var cancel context.CancelFunc
				ctx, cancel = tt.ctx()
				defer cancel()
			}

			resp, err := client.Refund(ctx, tt.req)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if resp != nil {
					t.Fatalf("expected nil response on error, got %+v", resp)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp == nil {
				t.Fatal("expected non-nil response")
			}
			if resp.Status != tt.wantStatus {
				t.Errorf("expected status %q, got %q", tt.wantStatus, resp.Status)
			}
			if resp.Success != tt.wantSuccess {
				t.Errorf("expected success %v, got %v", tt.wantSuccess, resp.Success)
			}
		})
	}
}

func TestClient_Concurrency(t *testing.T) {
	t.Parallel()

	client := New()
	ctx := context.Background()

	const workers = 20
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := 0; w < workers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				// Alternate operations
				switch (workerID + i) % 4 {
				case 0:
					resp, err := client.Charge(ctx, chargeReqSuccess())
					if err != nil || resp.Status != paykit.StatusSuccess {
						t.Errorf("concurrent charge failed: %v, status=%v", err, resp)
					}
				case 1:
					resp, err := client.Charge(ctx, chargeReqCancelled())
					if !errors.Is(err, ErrUserCancelled) || resp.Status != paykit.StatusFailed {
						t.Errorf("concurrent charge cancel failed: %v, status=%v", err, resp)
					}
				case 2:
					resp, err := client.Disburse(ctx, disburseReqSuccess())
					if err != nil || resp.Status != paykit.StatusSuccess {
						t.Errorf("concurrent disburse failed: %v, status=%v", err, resp)
					}
				case 3:
					resp, err := client.QueryStatus(ctx, buildStatusRequest("normal_id"))
					if err != nil || resp.Status != paykit.StatusSuccess {
						t.Errorf("concurrent query failed: %v, status=%v", err, resp)
					}
				}
			}
		}(w)
	}

	wg.Wait()
}
