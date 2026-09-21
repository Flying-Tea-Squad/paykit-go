package bogus

import (
	"errors"
	"testing"

	paykit "github.com/Flying-Tea-Squad/paykit-go"
)

// ── Name ─────────────────────────────────────────────────────────────────────

func TestClient_Name(t *testing.T) {
	c := newClient(t)
	if got := c.Name(); got != "bogus" {
		t.Errorf("Name() = %q; want %q", got, "bogus")
	}
}

// ── Purchase ──────────────────────────────────────────────────────────────────

func TestClient_Purchase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		amount    int
		wantOK    bool
		wantErr   error
		wantCode  string
	}{
		{
			name:    "ends in 00 — success",
			amount:  1000,
			wantOK:  true,
		},
		{
			name:    "exactly 0 — success",
			amount:  0,
			wantOK:  true,
		},
		{
			name:     "ends in 05 — card declined",
			amount:   1005,
			wantOK:   false,
			wantErr:  paykit.ErrCardDeclinedSentinel,
			wantCode: paykit.ErrCardDeclined,
		},
		{
			name:     "ends in 05 (small amount) — card declined",
			amount:   5,
			wantOK:   false,
			wantErr:  paykit.ErrCardDeclinedSentinel,
			wantCode: paykit.ErrCardDeclined,
		},
		{
			name:     "ends in other digits — processing error",
			amount:   1001,
			wantOK:   false,
			wantErr:  paykit.ErrProcessingErrorSentinel,
			wantCode: paykit.ErrProcessingError,
		},
	}

	c := newClient(t)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := c.Purchase(bg(), purchaseReq(tc.amount))
			assertOutcome(t, resp, err, tc.wantOK, tc.wantErr, tc.wantCode)
		})
	}
}

// ── Authorize ─────────────────────────────────────────────────────────────────

func TestClient_Authorize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		amount   int
		wantOK   bool
		wantErr  error
		wantCode string
	}{
		{"ends in 00 — success", 2000, true, nil, ""},
		{"ends in 05 — declined", 2005, false, paykit.ErrCardDeclinedSentinel, paykit.ErrCardDeclined},
		{"ends in 07 — processing error", 2007, false, paykit.ErrProcessingErrorSentinel, paykit.ErrProcessingError},
	}

	c := newClient(t)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := c.Authorize(bg(), authorizeReq(tc.amount))
			assertOutcome(t, resp, err, tc.wantOK, tc.wantErr, tc.wantCode)
		})
	}
}

// ── Capture ───────────────────────────────────────────────────────────────────

func TestClient_Capture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		txID     string
		amount   int
		wantOK   bool
		wantErr  error
		wantCode string
	}{
		{"ends in 00 — success", "tx-cap-ok", 3000, true, nil, ""},
		{"ends in 05 — declined", "tx-cap-fail", 3005, false, paykit.ErrCardDeclinedSentinel, paykit.ErrCardDeclined},
	}

	c := newClient(t)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := c.Capture(bg(), captureReq(tc.txID, tc.amount))
			assertOutcome(t, resp, err, tc.wantOK, tc.wantErr, tc.wantCode)
		})
	}
}

// ── Void ──────────────────────────────────────────────────────────────────────

func TestClient_Void(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		txID     string
		wantOK   bool
		wantErr  error
		wantCode string
	}{
		{"normal txID — success", "tx-void-ok", true, nil, ""},
		{"fail-prefixed txID — declined", "fail-tx-void", false, paykit.ErrCardDeclinedSentinel, paykit.ErrCardDeclined},
	}

	c := newClient(t)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := c.Void(bg(), voidReq(tc.txID))
			assertOutcome(t, resp, err, tc.wantOK, tc.wantErr, tc.wantCode)
		})
	}
}

// ── Refund ────────────────────────────────────────────────────────────────────

func TestClient_Refund(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		txID     string
		amount   int
		wantOK   bool
		wantErr  error
		wantCode string
	}{
		{"ends in 00 — success", "tx-ref-ok", 5000, true, nil, ""},
		{"ends in 05 — declined", "tx-ref-fail", 5005, false, paykit.ErrCardDeclinedSentinel, paykit.ErrCardDeclined},
	}

	c := newClient(t)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := c.Refund(bg(), refundReq(tc.txID, tc.amount))
			assertOutcome(t, resp, err, tc.wantOK, tc.wantErr, tc.wantCode)
		})
	}
}

// ── QueryStatus ───────────────────────────────────────────────────────────────

func TestClient_QueryStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		txID     string
		wantOK   bool
		wantErr  error
		wantCode string
	}{
		{"regular ID — success", "tx-123", true, nil, ""},
		{"fail-prefixed ID — declined", "fail-tx-123", false, paykit.ErrCardDeclinedSentinel, paykit.ErrCardDeclined},
		{"empty ID — success (zero value)", "", true, nil, ""},
	}

	c := newClient(t)
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := c.QueryStatus(bg(), statusReq(tc.txID))
			assertOutcome(t, resp, err, tc.wantOK, tc.wantErr, tc.wantCode)
		})
	}
}

// ── Registration ──────────────────────────────────────────────────────────────

// TestInit_Registration verifies the gateway is registered under "bogus" and
// that the factory returns a working *Client.
func TestInit_Registration(t *testing.T) {
	// Re-import gateway to exercise the registry.
	// The init() in client.go already ran; here we just verify the result.
	import_gateway(t)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func assertOutcome(
	t *testing.T,
	resp *paykit.Response,
	err error,
	wantOK bool,
	wantErr error,
	wantCode string,
) {
	t.Helper()

	if resp == nil {
		t.Fatal("response must not be nil")
	}
	if resp.Success != wantOK {
		t.Errorf("Response.Success = %v; want %v", resp.Success, wantOK)
	}
	if wantCode != "" && resp.ErrorCode != wantCode {
		t.Errorf("Response.ErrorCode = %q; want %q", resp.ErrorCode, wantCode)
	}
	if wantErr != nil {
		if !errors.Is(err, wantErr) {
			t.Errorf("err = %v; want %v", err, wantErr)
		}
	} else {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}
}
