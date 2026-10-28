package bogus

import "github.com/Flying-Tea-Squad/paykit-go"

func buildChargeRequest(amount int64, phone, ref string) *paykit.ChargeRequest {
	if phone == "" {
		phone = "+254712345678"
	}
	if ref == "" {
		ref = "REF123"
	}
	return &paykit.ChargeRequest{
		Amount:         paykit.NewMoney(amount, "KES"),
		Phone:          phone,
		Description:    "Test Charge",
		Reference:      ref,
		IdempotencyKey: "idem-" + ref,
	}
}

func buildDisbursementRequest(amount int64, phone, ref string) *paykit.DisbursementRequest {
	if phone == "" {
		phone = "+254798765432"
	}
	if ref == "" {
		ref = "DISB123"
	}
	return &paykit.DisbursementRequest{
		Amount:         paykit.NewMoney(amount, "KES"),
		Phone:          phone,
		RecipientName:  "Jane Doe",
		Description:    "Test Payout",
		Reference:      ref,
		IdempotencyKey: "idem-" + ref,
	}
}

func buildStatusRequest(txnID string) *paykit.StatusRequest {
	return &paykit.StatusRequest{
		TransactionID: txnID,
		Reference:     "REF_STATUS",
	}
}

func buildRefundRequest(txnID string, amount int64) *paykit.RefundRequest {
	return &paykit.RefundRequest{
		TransactionID:  txnID,
		Amount:         paykit.NewMoney(amount, "KES"),
		Reason:         "Customer dispute",
		IdempotencyKey: "idem-ref-" + txnID,
	}
}

// Presets for deterministic Charge scenarios
func chargeReqSuccess() *paykit.ChargeRequest {
	return buildChargeRequest(1000, "+254712000000", "SUCCESS_00")
}

func chargeReqCancelled() *paykit.ChargeRequest {
	return buildChargeRequest(1005, "+254712000005", "CANCEL_05")
}

func chargeReqPinTimeout() *paykit.ChargeRequest {
	return buildChargeRequest(1002, "+254712000002", "TIMEOUT_02")
}

func chargeReqProcessing() *paykit.ChargeRequest {
	return buildChargeRequest(1099, "+254712000099", "PROC_99")
}

// Presets for deterministic Disbursement scenarios
func disburseReqSuccess() *paykit.DisbursementRequest {
	return buildDisbursementRequest(2500, "+254798000000", "DISB_SUCCESS_00")
}

func disburseReqInsufficientFunds() *paykit.DisbursementRequest {
	return buildDisbursementRequest(2550, "+254798000050", "DISB_INSUFFICIENT_50")
}

func disburseReqProcessing() *paykit.DisbursementRequest {
	return buildDisbursementRequest(2525, "+254798000025", "DISB_PROC_25")
}
