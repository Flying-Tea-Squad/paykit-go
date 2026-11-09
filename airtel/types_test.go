package airtel

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCollectionRequestSerialization(t *testing.T) {
	req := CollectionRequest{
		Reference: "coll-ref-123",
		Subscriber: Subscriber{
			Country:  "TZ",
			Currency: "TZS",
			MSISDN:   "255712345678",
		},
		Transaction: CollectionTransaction{
			Amount:   "10000",
			Country:  "TZ",
			Currency: "TZS",
			ID:       "txn-001",
			Type:     "Collection",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal CollectionRequest: %v", err)
	}

	jsonStr := string(data)

	// Verify required fields are present
	requiredKeys := []string{
		"\"reference\"",
		"\"subscriber\"",
		"\"transaction\"",
		"\"country\"",
		"\"currency\"",
		"\"msisdn\"",
		"\"amount\"",
		"\"id\"",
		"\"type\"",
	}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("expected JSON to contain key %s, got: %s", key, jsonStr)
		}
	}

	// Verify PIN is omitted (not part of collection request)
	if strings.Contains(jsonStr, "\"pin\"") {
		t.Error("expected PIN to be absent from CollectionRequest")
	}

	// Verify amount serializes as string (major units)
	if !strings.Contains(jsonStr, "\"amount\":\"10000\"") {
		t.Errorf("expected amount as string in major units, got: %s", jsonStr)
	}
}

func TestCollectionResponseDeserialization(t *testing.T) {
	jsonInput := `{
		"status": {
			"code": "200",
			"message": "success",
			"success": true
		},
		"data": {
			"transaction": {
				"id": "txn-001",
				"status": "PENDING",
				"message": "USSD push sent",
				"statusCode": "00"
			}
		}
	}`

	var resp CollectionResponse
	if err := json.Unmarshal([]byte(jsonInput), &resp); err != nil {
		t.Fatalf("failed to unmarshal CollectionResponse: %v", err)
	}

	if resp.Status.Code != "200" {
		t.Errorf("expected status code 200, got %s", resp.Status.Code)
	}
	if resp.Status.Message != "success" {
		t.Errorf("expected status message success, got %s", resp.Status.Message)
	}
	if !resp.Status.Success {
		t.Error("expected status success true")
	}
	if resp.Data.Transaction.ID != "txn-001" {
		t.Errorf("expected transaction id txn-001, got %s", resp.Data.Transaction.ID)
	}
	if resp.Data.Transaction.Status != "PENDING" {
		t.Errorf("expected status PENDING, got %s", resp.Data.Transaction.Status)
	}
	if resp.Data.Transaction.Message != "USSD push sent" {
		t.Errorf("expected message USSD push sent, got %s", resp.Data.Transaction.Message)
	}
	if resp.Data.Transaction.StatusCode != "00" {
		t.Errorf("expected statusCode 00, got %s", resp.Data.Transaction.StatusCode)
	}
}

func TestDisbursementRequestSerialization(t *testing.T) {
	req := DisbursementRequest{
		Reference: "disb-ref-456",
		PIN:       "1234",
		Payee: Payee{
			Country:   "TZ",
			Currency:  "TZS",
			MSISDN:    "255712345678",
			FirstName: "John",
			LastName:  "Doe",
		},
		Transaction: DisbursementTransaction{
			Amount:   "50000",
			Country:  "TZ",
			Currency: "TZS",
			ID:       "txn-002",
			Type:     "Disbursement",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("failed to marshal DisbursementRequest: %v", err)
	}

	jsonStr := string(data)

	// Verify required fields are present
	requiredKeys := []string{
		"\"reference\"",
		"\"pin\"",
		"\"payee\"",
		"\"transaction\"",
		"\"country\"",
		"\"currency\"",
		"\"msisdn\"",
		"\"first_name\"",
		"\"last_name\"",
		"\"amount\"",
		"\"id\"",
		"\"type\"",
	}
	for _, key := range requiredKeys {
		if !strings.Contains(jsonStr, key) {
			t.Errorf("expected JSON to contain key %s, got: %s", key, jsonStr)
		}
	}

	// Verify subscriber is NOT present (should be payee)
	if strings.Contains(jsonStr, "\"subscriber\"") {
		t.Error("expected subscriber to be absent, payee should be used")
	}

	// Verify PIN is present and required
	if !strings.Contains(jsonStr, "\"pin\":\"1234\"") {
		t.Errorf("expected PIN to be present, got: %s", jsonStr)
	}
}

func TestDisbursementResponseDeserialization(t *testing.T) {
	jsonInput := `{
		"status": {
			"code": "200",
			"message": "success",
			"success": true
		},
		"data": {
			"transaction": {
				"id": "txn-002",
				"status": "SUCCESS",
				"statusCode": "00",
				"message": "Disbursement completed",
				"transactionId": "ATL123456"
			}
		}
	}`

	var resp DisbursementResponse
	if err := json.Unmarshal([]byte(jsonInput), &resp); err != nil {
		t.Fatalf("failed to unmarshal DisbursementResponse: %v", err)
	}

	if resp.Status.Code != "200" {
		t.Errorf("expected status code 200, got %s", resp.Status.Code)
	}
	if resp.Data.Transaction.ID != "txn-002" {
		t.Errorf("expected transaction id txn-002, got %s", resp.Data.Transaction.ID)
	}
	if resp.Data.Transaction.Status != "SUCCESS" {
		t.Errorf("expected status SUCCESS, got %s", resp.Data.Transaction.Status)
	}
	if resp.Data.Transaction.TransactionID != "ATL123456" {
		t.Errorf("expected transactionId ATL123456, got %s", resp.Data.Transaction.TransactionID)
	}
}

func TestTransactionStatusResponseDeserialization(t *testing.T) {
	jsonInput := `{
		"status": {
			"code": "200",
			"message": "success",
			"success": true
		},
		"data": {
			"transaction": {
				"airtel_money_id": "ATL789012",
				"id": "txn-003",
				"status": "COMPLETED",
				"message": "Transaction completed successfully",
				"statusCode": "00"
			}
		}
	}`

	var resp TransactionStatusResponse
	if err := json.Unmarshal([]byte(jsonInput), &resp); err != nil {
		t.Fatalf("failed to unmarshal TransactionStatusResponse: %v", err)
	}

	if resp.Status.Code != "200" {
		t.Errorf("expected status code 200, got %s", resp.Status.Code)
	}
	if resp.Data.Transaction.AirtelMoneyID != "ATL789012" {
		t.Errorf("expected airtel_money_id ATL789012, got %s", resp.Data.Transaction.AirtelMoneyID)
	}
	if resp.Data.Transaction.ID != "txn-003" {
		t.Errorf("expected id txn-003, got %s", resp.Data.Transaction.ID)
	}
	if resp.Data.Transaction.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", resp.Data.Transaction.Status)
	}
	if resp.Data.Transaction.Message != "Transaction completed successfully" {
		t.Errorf("expected message Transaction completed successfully, got %s", resp.Data.Transaction.Message)
	}
}

func TestReconciliationResponseDeserialization(t *testing.T) {
	jsonInput := `{
		"status": {
			"code": "200",
			"message": "success",
			"success": true
		},
		"data": {
			"transactions": [
				{
					"id": "txn-001",
					"reference": "ref-001",
					"status": "SUCCESS",
					"statusCode": "00",
					"amount": "10000",
					"currency": "TZS",
					"msisdn": "255712345678",
					"timestamp": "2024-01-15T10:30:00Z",
					"type": "Collection"
				},
				{
					"id": "txn-002",
					"reference": "ref-002",
					"status": "FAILED",
					"statusCode": "02",
					"amount": "50000",
					"currency": "TZS",
					"msisdn": "255712345678",
					"timestamp": "2024-01-15T11:00:00Z",
					"type": "Disbursement"
				}
			],
			"summary": {
				"totalCount": 2,
				"totalAmount": "60000",
				"currency": "TZS",
				"successfulCount": 1,
				"failedCount": 1,
				"pendingCount": 0
			}
		}
	}`

	var resp ReconciliationResponse
	if err := json.Unmarshal([]byte(jsonInput), &resp); err != nil {
		t.Fatalf("failed to unmarshal ReconciliationResponse: %v", err)
	}

	if resp.Status.Code != "200" {
		t.Errorf("expected status code 200, got %s", resp.Status.Code)
	}
	if len(resp.Data.Transactions) != 2 {
		t.Errorf("expected 2 transactions, got %d", len(resp.Data.Transactions))
	}
	if resp.Data.Transactions[0].ID != "txn-001" {
		t.Errorf("expected first transaction id txn-001, got %s", resp.Data.Transactions[0].ID)
	}
	if resp.Data.Transactions[1].Type != "Disbursement" {
		t.Errorf("expected second transaction type Disbursement, got %s", resp.Data.Transactions[1].Type)
	}
	if resp.Data.Summary.TotalCount != 2 {
		t.Errorf("expected totalCount 2, got %d", resp.Data.Summary.TotalCount)
	}
	if resp.Data.Summary.SuccessfulCount != 1 {
		t.Errorf("expected successfulCount 1, got %d", resp.Data.Summary.SuccessfulCount)
	}
}
