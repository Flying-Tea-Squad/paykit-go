package mpesa

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAccountBalanceSerialization(t *testing.T) {
	req := BalanceRequest{
		IdempotencyKey:  "balance-key-999",
		Initiator:       "testapiuser",
		CommandID:       "AccountBalance",
		PartyA:          "600000",
		IdentifierType:  "4",
		Remarks:         "reconciliation check",
		QueueTimeOutURL: "http://myservice:8080/queuetimeouturl",
		ResultURL:       "http://myservice:8080/result",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal BalanceRequest: %v", err)
	}

	jsonStr := string(data)

	// Ensure our routing metadata (IdempotencyKey) is omitted from the JSON payload
	if strings.Contains(jsonStr, "IdempotencyKey") {
		t.Error("expected IdempotencyKey to be omitted from BalanceRequest body text formatting")
	}

	// Verify crucial PascalCase key transformations required by Daraja
	requiredKeys := []string{"\"Initiator\"", "\"CommandID\"", "\"PartyA\"", "\"IdentifierType\"", "\"ResultURL\""}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("expected JSON to possess target key %s, but got: %s", key, jsonStr)
		}
	}
}

func TestQRCodeSerialization(t *testing.T) {
	req := DynamicQRRequest{
		IdempotencyKey: "qr-gen-771",
		MerchantName:   "TEST SUPERMARKET",
		RefNo:          "Invoice Test",
		Amount:         1,
		TrxCode:        "BG",
		CPI:            "373132",
		Size:           "300",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal QRRequest: %v", err)
	}

	jsonStr := string(data)

	// Ensure our routing metadata (IdempotencyKey) is omitted from the JSON payload
	if strings.Contains(jsonStr, "IdempotencyKey") {
		t.Error("expected IdempotencyKey to be omitted from QRRequest body text formatting")
	}

	// Confirm that Amount serializes strictly as a raw numeric unquoted integer value
	if !strings.Contains(jsonStr, "\"Amount\":1") {
		t.Errorf("expected numeric structure formatting for Amount field, but got: %s", jsonStr)
	}

	// Verify crucial PascalCase keys required by Daraja's endpoint schema layout
	requiredKeys := []string{"\"MerchantName\"", "\"RefNo\"", "\"TrxCode\"", "\"CPI\"", "\"Size\""}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("expected JSON to possess target key %s, but got: %s", key, jsonStr)
		}
	}
}

func TestC2BRegisterRequestSerialization(t *testing.T) {
	req := C2BRegisterRequest{
		IdempotencyKey:  "c2b-reg-key-999",
		ShortCode:       "600000",
		ResponseType:    "Completed",
		ConfirmationURL: "https://example.com/confirmation",
		ValidationURL:   "https://example.com/validation",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal C2BRegisterRequest: %v", err)
	}

	jsonStr := string(data)

	// Ensure our routing metadata (IdempotencyKey) is omitted from the JSON payload
	if strings.Contains(jsonStr, "IdempotencyKey") {
		t.Error("expected IdempotencyKey to be omitted from C2BRegisterRequest body")
	}

	// Verify PascalCase keys required by Daraja
	requiredKeys := []string{"\"ShortCode\"", "\"ResponseType\"", "\"ConfirmationURL\"", "\"ValidationURL\""}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("expected JSON to possess target key %s, but got: %s", key, jsonStr)
		}
	}
}

func TestC2BSimulateRequestSerialization(t *testing.T) {
	req := C2BSimulateRequest{
		IdempotencyKey: "c2b-sim-key-999",
		ShortCode:      "600000",
		CommandID:      "CustomerPayBillOnline",
		Amount:         100,
		Msisdn:         "254700000001",
		BillRefNumber:  "INV001",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal C2BSimulateRequest: %v", err)
	}

	jsonStr := string(data)

	// Ensure our routing metadata (IdempotencyKey) is omitted from the JSON payload
	if strings.Contains(jsonStr, "IdempotencyKey") {
		t.Error("expected IdempotencyKey to be omitted from C2BSimulateRequest body")
	}

	// Confirm numeric Amount serialization
	if !strings.Contains(jsonStr, "\"Amount\":100") {
		t.Errorf("expected numeric structure formatting for Amount field, but got: %s", jsonStr)
	}

	// Verify PascalCase keys required by Daraja
	requiredKeys := []string{"\"ShortCode\"", "\"CommandID\"", "\"Amount\"", "\"Msisdn\"", "\"BillRefNumber\""}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("expected JSON to possess target key %s, but got: %s", key, jsonStr)
		}
	}
}
