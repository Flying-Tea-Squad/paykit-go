package bogus_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Flying-Tea-Squad/paykit-go"
	"github.com/Flying-Tea-Squad/paykit-go/bogus"
)

func TestGateway_RegistryDiscovery(t *testing.T) {
	t.Parallel()

	gw, err := paykit.Get("bogus")
	if err != nil {
		t.Fatalf("expected paykit.Get('bogus') to succeed, got: %v", err)
	}

	if gw == nil {
		t.Fatal("expected non-nil gateway instance")
	}

	if gw.Name() != "bogus" {
		t.Fatalf("expected gateway name 'bogus', got %q", gw.Name())
	}

	disburser, ok := gw.(paykit.Disburser)
	if !ok || disburser == nil {
		t.Fatalf("expected registered gateway to satisfy paykit.Disburser capability")
	}
}

func TestGateway_Constructor(t *testing.T) {
	t.Parallel()

	client := bogus.New()
	if client == nil {
		t.Fatal("expected bogus.New() to return non-nil Client")
	}

	if client.Name() != "bogus" {
		t.Fatalf("expected client name 'bogus', got %q", client.Name())
	}

	var _ paykit.Gateway = client
	var _ paykit.Disburser = client
}

func TestGateway_Charge_PublicContract(t *testing.T) {
	t.Parallel()

	gw, err := paykit.Get("bogus")
	if err != nil {
		t.Fatalf("failed to resolve bogus gateway: %v", err)
	}

	ctx := context.Background()

	t.Run("success scenario via SuffixSuccess", func(t *testing.T) {
		t.Parallel()
		req := &paykit.ChargeRequest{
			Amount:      paykit.NewMoney(1000+bogus.SuffixSuccess, "KES"),
			Phone:       "+254712000000",
			Description: "Test Charge Success",
			Reference:   "BB_CHG_00",
		}

		resp, err := gw.Charge(ctx, req)
		if err != nil {
			t.Fatalf("unexpected charge error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil charge response")
		}
		if !resp.Success {
			t.Errorf("expected resp.Success to be true, got false")
		}
		if resp.Status != paykit.StatusSuccess {
			t.Errorf("expected status %q, got %q", paykit.StatusSuccess, resp.Status)
		}
		if !strings.HasPrefix(resp.TransactionID, "bogus_chg_") {
			t.Errorf("expected transaction ID prefix 'bogus_chg_', got %q", resp.TransactionID)
		}
	})

	t.Run("customer cancelled scenario via SuffixUserCancelled", func(t *testing.T) {
		t.Parallel()
		req := &paykit.ChargeRequest{
			Amount:      paykit.NewMoney(1000+bogus.SuffixUserCancelled, "KES"),
			Phone:       "+254712000005",
			Description: "Test Charge Cancelled",
			Reference:   "BB_CHG_05",
		}

		resp, err := gw.Charge(ctx, req)
		if err != nil {
			t.Fatalf("expected nil error for business decline, got: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil charge response")
		}
		if resp.Success {
			t.Errorf("expected resp.Success to be false, got true")
		}
		if resp.Status != paykit.StatusCancelled {
			t.Errorf("expected status %q, got %q", paykit.StatusCancelled, resp.Status)
		}
		if !strings.HasPrefix(resp.TransactionID, "fail_chg_") {
			t.Errorf("expected transaction ID prefix 'fail_chg_', got %q", resp.TransactionID)
		}
	})

	t.Run("pin timeout scenario via SuffixPinTimeout", func(t *testing.T) {
		t.Parallel()
		req := &paykit.ChargeRequest{
			Amount:      paykit.NewMoney(1000+bogus.SuffixPinTimeout, "KES"),
			Phone:       "+254712000002",
			Description: "Test Charge Timeout",
			Reference:   "BB_CHG_02",
		}

		resp, err := gw.Charge(ctx, req)
		if err != nil {
			t.Fatalf("expected nil error for business decline, got: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil charge response")
		}
		if resp.Success {
			t.Errorf("expected resp.Success to be false, got true")
		}
		if resp.Status != paykit.StatusTimeout {
			t.Errorf("expected status %q, got %q", paykit.StatusTimeout, resp.Status)
		}
		if !strings.HasPrefix(resp.TransactionID, "fail_chg_") {
			t.Errorf("expected transaction ID prefix 'fail_chg_', got %q", resp.TransactionID)
		}
	})

	t.Run("processing fallback scenario", func(t *testing.T) {
		t.Parallel()
		req := &paykit.ChargeRequest{
			Amount:      paykit.NewMoney(1099, "KES"),
			Phone:       "+254712000099",
			Description: "Test Charge Processing",
			Reference:   "BB_CHG_99",
		}

		resp, err := gw.Charge(ctx, req)
		if err != nil {
			t.Fatalf("unexpected charge error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil charge response")
		}
		if !resp.Success {
			t.Errorf("expected resp.Success to be true, got false")
		}
		if resp.Status != paykit.StatusProcessing {
			t.Errorf("expected status %q, got %q", paykit.StatusProcessing, resp.Status)
		}
	})

	t.Run("zero amount returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.ChargeRequest{
			Amount: paykit.NewMoney(0, "KES"),
			Phone:  "+254712000000",
		}

		resp, err := gw.Charge(ctx, req)
		if err == nil {
			t.Fatal("expected error for zero amount, got nil")
		}
		if !errors.Is(err, bogus.ErrChargeAmountInvalid) {
			t.Errorf("expected ErrChargeAmountInvalid, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.ChargeRequest{
			Amount: paykit.NewMoney(-500, "KES"),
			Phone:  "+254712000000",
		}

		resp, err := gw.Charge(ctx, req)
		if err == nil {
			t.Fatal("expected error for negative amount, got nil")
		}
		if !errors.Is(err, bogus.ErrChargeAmountInvalid) {
			t.Errorf("expected ErrChargeAmountInvalid, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("nil request returns ErrNilRequest", func(t *testing.T) {
		t.Parallel()
		resp, err := gw.Charge(ctx, nil)
		if !errors.Is(err, bogus.ErrNilRequest) {
			t.Fatalf("expected ErrNilRequest, got: %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on error, got %+v", resp)
		}
	})
}

func TestGateway_Disburse_PublicContract(t *testing.T) {
	t.Parallel()

	gw, err := paykit.Get("bogus")
	if err != nil {
		t.Fatalf("failed to resolve bogus gateway: %v", err)
	}

	disburser, ok := gw.(paykit.Disburser)
	if !ok {
		t.Fatalf("gateway does not implement paykit.Disburser")
	}

	ctx := context.Background()

	t.Run("success disbursement via SuffixSuccess", func(t *testing.T) {
		t.Parallel()
		req := &paykit.DisbursementRequest{
			Amount:        paykit.NewMoney(2000+bogus.SuffixSuccess, "KES"),
			Phone:         "+254798000000",
			RecipientName: "John Doe",
			Reference:     "BB_DISB_00",
		}

		resp, err := disburser.Disburse(ctx, req)
		if err != nil {
			t.Fatalf("unexpected disbursement error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if !resp.Success {
			t.Errorf("expected resp.Success to be true, got false")
		}
		if resp.Status != paykit.StatusSuccess {
			t.Errorf("expected status %q, got %q", paykit.StatusSuccess, resp.Status)
		}
		if !strings.HasPrefix(resp.TransactionID, "bogus_disb_") {
			t.Errorf("expected transaction ID prefix 'bogus_disb_', got %q", resp.TransactionID)
		}
	})

	t.Run("insufficient funds scenario via SuffixInsufficientFunds", func(t *testing.T) {
		t.Parallel()
		req := &paykit.DisbursementRequest{
			Amount:        paykit.NewMoney(2000+bogus.SuffixInsufficientFunds, "KES"),
			Phone:         "+254798000050",
			RecipientName: "John Doe",
			Reference:     "BB_DISB_50",
		}

		resp, err := disburser.Disburse(ctx, req)
		if err != nil {
			t.Fatalf("expected nil error for business decline, got: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if resp.Success {
			t.Errorf("expected resp.Success to be false, got true")
		}
		if resp.Status != paykit.StatusFailed {
			t.Errorf("expected status %q, got %q", paykit.StatusFailed, resp.Status)
		}
		if !strings.HasPrefix(resp.TransactionID, "fail_disb_") {
			t.Errorf("expected transaction ID prefix 'fail_disb_', got %q", resp.TransactionID)
		}
	})

	t.Run("processing fallback scenario", func(t *testing.T) {
		t.Parallel()
		req := &paykit.DisbursementRequest{
			Amount:        paykit.NewMoney(2025, "KES"),
			Phone:         "+254798000025",
			RecipientName: "John Doe",
			Reference:     "BB_DISB_25",
		}

		resp, err := disburser.Disburse(ctx, req)
		if err != nil {
			t.Fatalf("unexpected disbursement error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil response")
		}
		if !resp.Success {
			t.Errorf("expected resp.Success to be true, got false")
		}
		if resp.Status != paykit.StatusProcessing {
			t.Errorf("expected status %q, got %q", paykit.StatusProcessing, resp.Status)
		}
	})

	t.Run("disbursement fallback transaction reference format", func(t *testing.T) {
		t.Parallel()
		req := &paykit.DisbursementRequest{
			Amount: paykit.NewMoney(1500, "KES"),
			Phone:  "+254798000000",
		}

		resp, err := disburser.Disburse(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.TransactionID != "bogus_disb_txn_1500" {
			t.Errorf("expected transaction ID 'bogus_disb_txn_1500', got %q", resp.TransactionID)
		}
	})

	t.Run("zero amount returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.DisbursementRequest{
			Amount: paykit.NewMoney(0, "KES"),
			Phone:  "+254798000000",
		}

		resp, err := disburser.Disburse(ctx, req)
		if err == nil {
			t.Fatal("expected error for zero amount, got nil")
		}
		if !errors.Is(err, bogus.ErrDisburseAmountInvalid) {
			t.Errorf("expected ErrDisburseAmountInvalid, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("negative amount returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.DisbursementRequest{
			Amount: paykit.NewMoney(-1000, "KES"),
			Phone:  "+254798000000",
		}

		resp, err := disburser.Disburse(ctx, req)
		if err == nil {
			t.Fatal("expected error for negative amount, got nil")
		}
		if !errors.Is(err, bogus.ErrDisburseAmountInvalid) {
			t.Errorf("expected ErrDisburseAmountInvalid, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("nil request returns ErrNilRequest", func(t *testing.T) {
		t.Parallel()
		resp, err := disburser.Disburse(ctx, nil)
		if !errors.Is(err, bogus.ErrNilRequest) {
			t.Fatalf("expected ErrNilRequest, got: %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on error, got %+v", resp)
		}
	})
}

func TestGateway_QueryStatus_PublicContract(t *testing.T) {
	t.Parallel()

	gw, err := paykit.Get("bogus")
	if err != nil {
		t.Fatalf("failed to resolve bogus gateway: %v", err)
	}

	ctx := context.Background()

	t.Run("success status query", func(t *testing.T) {
		t.Parallel()
		req := &paykit.StatusRequest{
			TransactionID: "bogus_chg_success_123",
			Reference:     "BB_STATUS_01",
		}

		resp, err := gw.QueryStatus(ctx, req)
		if err != nil {
			t.Fatalf("unexpected query status error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil status response")
		}
		if !resp.Success {
			t.Errorf("expected resp.Success to be true, got false")
		}
		if resp.Status != paykit.StatusSuccess {
			t.Errorf("expected status %q, got %q", paykit.StatusSuccess, resp.Status)
		}
		if resp.TransactionID != req.TransactionID {
			t.Errorf("expected transaction ID %q, got %q", req.TransactionID, resp.TransactionID)
		}
	})

	t.Run("failed status query", func(t *testing.T) {
		t.Parallel()
		req := &paykit.StatusRequest{
			TransactionID: "fail_chg_timeout_456",
			Reference:     "BB_STATUS_02",
		}

		resp, err := gw.QueryStatus(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error for business status: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil status response")
		}
		if resp.Success {
			t.Errorf("expected resp.Success to be false, got true")
		}
		if resp.Status != paykit.StatusFailed {
			t.Errorf("expected status %q, got %q", paykit.StatusFailed, resp.Status)
		}
	})

	t.Run("empty transaction ID returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.StatusRequest{TransactionID: ""}

		resp, err := gw.QueryStatus(ctx, req)
		if err == nil {
			t.Fatal("expected error for empty transaction ID, got nil")
		}
		if !errors.Is(err, bogus.ErrTransactionIDRequired) {
			t.Errorf("expected ErrTransactionIDRequired, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("whitespace transaction ID returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.StatusRequest{TransactionID: "   "}

		resp, err := gw.QueryStatus(ctx, req)
		if err == nil {
			t.Fatal("expected error for whitespace transaction ID, got nil")
		}
		if !errors.Is(err, bogus.ErrTransactionIDRequired) {
			t.Errorf("expected ErrTransactionIDRequired, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("nil request returns ErrNilRequest", func(t *testing.T) {
		t.Parallel()
		resp, err := gw.QueryStatus(ctx, nil)
		if !errors.Is(err, bogus.ErrNilRequest) {
			t.Fatalf("expected ErrNilRequest, got: %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on error, got %+v", resp)
		}
	})
}

func TestGateway_Refund_PublicContract(t *testing.T) {
	t.Parallel()

	gw, err := paykit.Get("bogus")
	if err != nil {
		t.Fatalf("failed to resolve bogus gateway: %v", err)
	}

	ctx := context.Background()

	t.Run("success refund", func(t *testing.T) {
		t.Parallel()
		req := &paykit.RefundRequest{
			TransactionID:  "txn_charge_to_refund",
			Amount:         paykit.NewMoney(1000, "KES"),
			Reason:         "Customer returned item",
			IdempotencyKey: "idem-refund-01",
		}

		resp, err := gw.Refund(ctx, req)
		if err != nil {
			t.Fatalf("unexpected refund error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil refund response")
		}
		if !resp.Success {
			t.Errorf("expected resp.Success to be true, got false")
		}
		if resp.Status != paykit.StatusSuccess {
			t.Errorf("expected status %q, got %q", paykit.StatusSuccess, resp.Status)
		}
		if !strings.HasPrefix(resp.RefundID, "bogus_ref_") {
			t.Errorf("expected refund ID prefix 'bogus_ref_', got %q", resp.RefundID)
		}
	})

	t.Run("failed refund", func(t *testing.T) {
		t.Parallel()
		req := &paykit.RefundRequest{
			TransactionID:  "fail_charge_original",
			Amount:         paykit.NewMoney(1000, "KES"),
			Reason:         "Failed reversal attempt",
			IdempotencyKey: "idem-refund-02",
		}

		resp, err := gw.Refund(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error for business decline: %v", err)
		}
		if resp == nil {
			t.Fatal("expected non-nil refund response")
		}
		if resp.Success {
			t.Errorf("expected resp.Success to be false, got true")
		}
		if resp.Status != paykit.StatusFailed {
			t.Errorf("expected status %q, got %q", paykit.StatusFailed, resp.Status)
		}
		if !strings.HasPrefix(resp.RefundID, "fail_ref_") {
			t.Errorf("expected refund ID prefix 'fail_ref_', got %q", resp.RefundID)
		}
	})

	t.Run("empty transaction ID returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.RefundRequest{TransactionID: ""}

		resp, err := gw.Refund(ctx, req)
		if err == nil {
			t.Fatal("expected error for empty transaction ID, got nil")
		}
		if !errors.Is(err, bogus.ErrTransactionIDRequired) {
			t.Errorf("expected ErrTransactionIDRequired, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("whitespace transaction ID returns error", func(t *testing.T) {
		t.Parallel()
		req := &paykit.RefundRequest{TransactionID: "   "}

		resp, err := gw.Refund(ctx, req)
		if err == nil {
			t.Fatal("expected error for whitespace transaction ID, got nil")
		}
		if !errors.Is(err, bogus.ErrTransactionIDRequired) {
			t.Errorf("expected ErrTransactionIDRequired, got %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on validation error, got %+v", resp)
		}
	})

	t.Run("nil request returns ErrNilRequest", func(t *testing.T) {
		t.Parallel()
		resp, err := gw.Refund(ctx, nil)
		if !errors.Is(err, bogus.ErrNilRequest) {
			t.Fatalf("expected ErrNilRequest, got: %v", err)
		}
		if resp != nil {
			t.Errorf("expected nil response on error, got %+v", resp)
		}
	})
}
