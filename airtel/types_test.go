package airtel

import (
	"encoding/json"
	"testing"
)

func TestCollectionRequestSerialization(t *testing.T) {
	tests := []struct {
		name          string
		req           CollectionRequest
		wantKeys      []string
		unwantKeys    []string
		wantAmountStr string
	}{
		{
			name: "basic",
			req: CollectionRequest{
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
			},
			wantKeys: []string{
				"\"reference\"", "\"subscriber\"", "\"transaction\"",
				"\"country\"", "\"currency\"", "\"msisdn\"",
				"\"amount\"", "\"id\"", "\"type\"",
			},
			unwantKeys:    []string{"\"pin\""},
			wantAmountStr: "\"amount\":\"10000\"",
		},
		{
			name: "different country",
			req: CollectionRequest{
				Reference: "coll-ref-456",
				Subscriber: Subscriber{
					Country:  "KE",
					Currency: "KES",
					MSISDN:   "254712345678",
				},
				Transaction: CollectionTransaction{
					Amount:   "500.50",
					Country:  "KE",
					Currency: "KES",
					ID:       "txn-002",
					Type:     "Collection",
				},
			},
			wantKeys: []string{
				"\"reference\"", "\"subscriber\"", "\"transaction\"",
				"\"country\"", "\"currency\"", "\"msisdn\"",
				"\"amount\"", "\"id\"", "\"type\"",
			},
			unwantKeys:    []string{"\"pin\""},
			wantAmountStr: "\"amount\":\"500.50\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal CollectionRequest: %v", err)
			}

			jsonStr := string(data)

			for _, key := range tt.wantKeys {
				if !contains(jsonStr, key) {
					t.Errorf("expected JSON to contain key %s, got: %s", key, jsonStr)
				}
			}

			for _, key := range tt.unwantKeys {
				if contains(jsonStr, key) {
					t.Errorf("expected JSON to NOT contain key %s, got: %s", key, jsonStr)
				}
			}

			if tt.wantAmountStr != "" && !contains(jsonStr, tt.wantAmountStr) {
				t.Errorf("expected amount string %s, got: %s", tt.wantAmountStr, jsonStr)
			}
		})
	}
}

func TestCollectionResponseRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantCode  string
		wantID    string
		wantMsg   string
	}{
		{
			name: "pending with message",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {"transaction": {"id": "txn-001", "status": "PENDING", "message": "USSD push sent", "statusCode": "00"}}
			}`,
			wantCode: "200",
			wantID:   "txn-001",
			wantMsg:  "USSD push sent",
		},
		{
			name: "completed without message",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {"transaction": {"id": "txn-002", "status": "COMPLETED", "statusCode": "00"}}
			}`,
			wantCode: "200",
			wantID:   "txn-002",
			wantMsg:  "",
		},
		{
			name: "failed with status code",
			jsonInput: `{
				"status": {"code": "400", "message": "failed", "success": false},
				"data": {"transaction": {"id": "txn-003", "status": "FAILED", "message": "Insufficient funds", "statusCode": "02"}}
			}`,
			wantCode: "400",
			wantID:   "txn-003",
			wantMsg:  "Insufficient funds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp CollectionResponse
			if err := json.Unmarshal([]byte(tt.jsonInput), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if resp.Status.Code != tt.wantCode {
				t.Errorf("status code: got %s, want %s", resp.Status.Code, tt.wantCode)
			}
			if resp.Data.Transaction.ID != tt.wantID {
				t.Errorf("transaction id: got %s, want %s", resp.Data.Transaction.ID, tt.wantID)
			}
			if resp.Data.Transaction.Message != tt.wantMsg {
				t.Errorf("message: got %s, want %s", resp.Data.Transaction.Message, tt.wantMsg)
			}

			// Marshal back and compare as JSON (key-order independent)
			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal back: %v", err)
			}

			var original, roundtrip map[string]any
			if err := json.Unmarshal([]byte(tt.jsonInput), &original); err != nil {
				t.Fatalf("failed to unmarshal original: %v", err)
			}
			if err := json.Unmarshal(data, &roundtrip); err != nil {
				t.Fatalf("failed to unmarshal roundtrip: %v", err)
			}
			if !jsonEqual(original, roundtrip) {
				t.Errorf("roundtrip JSON differs\noriginal: %s\nroundtrip: %s", tt.jsonInput, string(data))
			}
		})
	}
}

func TestDisbursementRequestSerialization(t *testing.T) {
	tests := []struct {
		name       string
		req        DisbursementRequest
		wantKeys   []string
		unwantKeys []string
	}{
		{
			name: "with payee names",
			req: DisbursementRequest{
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
			},
			wantKeys: []string{
				"\"reference\"", "\"pin\"", "\"payee\"", "\"transaction\"",
				"\"country\"", "\"currency\"", "\"msisdn\"",
				"\"first_name\"", "\"last_name\"",
				"\"amount\"", "\"id\"", "\"type\"",
			},
			unwantKeys: []string{"\"subscriber\""},
		},
		{
			name: "without payee names (omitempty)",
			req: DisbursementRequest{
				Reference: "disb-ref-789",
				PIN:       "5678",
				Payee: Payee{
					Country:  "KE",
					Currency: "KES",
					MSISDN:   "254712345678",
				},
				Transaction: DisbursementTransaction{
					Amount:   "1000",
					Country:  "KE",
					Currency: "KES",
					ID:       "txn-003",
					Type:     "Disbursement",
				},
			},
			wantKeys: []string{
				"\"reference\"", "\"pin\"", "\"payee\"", "\"transaction\"",
				"\"country\"", "\"currency\"", "\"msisdn\"",
				"\"amount\"", "\"id\"", "\"type\"",
			},
			unwantKeys: []string{"\"subscriber\"", "\"first_name\"", "\"last_name\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal DisbursementRequest: %v", err)
			}

			jsonStr := string(data)

			for _, key := range tt.wantKeys {
				if !contains(jsonStr, key) {
					t.Errorf("expected JSON to contain key %s, got: %s", key, jsonStr)
				}
			}

			for _, key := range tt.unwantKeys {
				if contains(jsonStr, key) {
					t.Errorf("expected JSON to NOT contain key %s, got: %s", key, jsonStr)
				}
			}

			if !contains(jsonStr, "\"pin\":\"") {
				t.Errorf("expected PIN to be present, got: %s", jsonStr)
			}
		})
	}
}

func TestDisbursementResponseRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantCode  string
		wantID    string
		wantTxnID string
	}{
		{
			name: "success with transactionId",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {"transaction": {"id": "txn-002", "status": "SUCCESS", "statusCode": "00", "message": "Disbursement completed", "transactionId": "ATL123456"}}
			}`,
			wantCode:  "200",
			wantID:    "txn-002",
			wantTxnID: "ATL123456",
		},
		{
			name: "pending without transactionId",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {"transaction": {"id": "txn-004", "status": "PENDING", "statusCode": "00"}}
			}`,
			wantCode:  "200",
			wantID:    "txn-004",
			wantTxnID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp DisbursementResponse
			if err := json.Unmarshal([]byte(tt.jsonInput), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if resp.Status.Code != tt.wantCode {
				t.Errorf("status code: got %s, want %s", resp.Status.Code, tt.wantCode)
			}
			if resp.Data.Transaction.ID != tt.wantID {
				t.Errorf("transaction id: got %s, want %s", resp.Data.Transaction.ID, tt.wantID)
			}
			if resp.Data.Transaction.TransactionID != tt.wantTxnID {
				t.Errorf("transactionId: got %s, want %s", resp.Data.Transaction.TransactionID, tt.wantTxnID)
			}

			// Marshal back and compare
			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal back: %v", err)
			}
			if !jsonEqualString(tt.jsonInput, string(data)) {
				t.Errorf("roundtrip JSON differs\noriginal: %s\nroundtrip: %s", tt.jsonInput, string(data))
			}
		})
	}
}

func TestTransactionStatusResponseRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		jsonInput string
		wantCode  string
		wantID    string
		wantAMLID string
		wantMsg   string
	}{
		{
			name: "with airtel_money_id and message",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {"transaction": {"airtel_money_id": "ATL789012", "id": "txn-003", "status": "COMPLETED", "message": "Transaction completed successfully", "statusCode": "00"}}
			}`,
			wantCode:  "200",
			wantID:    "txn-003",
			wantAMLID: "ATL789012",
			wantMsg:   "Transaction completed successfully",
		},
		{
			name: "without airtel_money_id",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {"transaction": {"id": "txn-005", "status": "PENDING", "statusCode": "00"}}
			}`,
			wantCode:  "200",
			wantID:    "txn-005",
			wantAMLID: "",
			wantMsg:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp TransactionStatusResponse
			if err := json.Unmarshal([]byte(tt.jsonInput), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if resp.Status.Code != tt.wantCode {
				t.Errorf("status code: got %s, want %s", resp.Status.Code, tt.wantCode)
			}
			if resp.Data.Transaction.ID != tt.wantID {
				t.Errorf("transaction id: got %s, want %s", resp.Data.Transaction.ID, tt.wantID)
			}
			if resp.Data.Transaction.AirtelMoneyID != tt.wantAMLID {
				t.Errorf("airtel_money_id: got %s, want %s", resp.Data.Transaction.AirtelMoneyID, tt.wantAMLID)
			}
			if resp.Data.Transaction.Message != tt.wantMsg {
				t.Errorf("message: got %s, want %s", resp.Data.Transaction.Message, tt.wantMsg)
			}

			// Marshal back and compare
			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal back: %v", err)
			}
			if !jsonEqualString(tt.jsonInput, string(data)) {
				t.Errorf("roundtrip JSON differs\noriginal: %s\nroundtrip: %s", tt.jsonInput, string(data))
			}
		})
	}
}

func TestReconciliationRequestSerialization(t *testing.T) {
	tests := []struct {
		name       string
		req        ReconciliationRequest
		wantKeys   []string
		unwantKeys []string
	}{
		{
			name: "with all optional filters",
			req: ReconciliationRequest{
				FromDate:      "2024-01-01T00:00:00Z",
				ToDate:        "2024-01-31T23:59:59Z",
				Reference:     "ref-123",
				TransactionID: "txn-456",
			},
			wantKeys:   []string{"\"fromDate\"", "\"toDate\"", "\"reference\"", "\"transactionId\""},
			unwantKeys: []string{},
		},
		{
			name: "without optional filters (omitempty)",
			req: ReconciliationRequest{
				FromDate: "2024-01-01T00:00:00Z",
				ToDate:   "2024-01-31T23:59:59Z",
			},
			wantKeys:   []string{"\"fromDate\"", "\"toDate\""},
			unwantKeys: []string{"\"reference\"", "\"transactionId\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("failed to marshal ReconciliationRequest: %v", err)
			}

			jsonStr := string(data)

			for _, key := range tt.wantKeys {
				if !contains(jsonStr, key) {
					t.Errorf("expected JSON to contain key %s, got: %s", key, jsonStr)
				}
			}

			for _, key := range tt.unwantKeys {
				if contains(jsonStr, key) {
					t.Errorf("expected JSON to NOT contain key %s, got: %s", key, jsonStr)
				}
			}
		})
	}
}

func TestReconciliationResponseRoundtrip(t *testing.T) {
	tests := []struct {
		name        string
		jsonInput   string
		wantCode    string
		wantTxnCnt  int
		wantSummary bool
	}{
		{
			name: "with transactions and summary",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {
					"transactions": [
						{"id": "txn-001", "reference": "ref-001", "status": "SUCCESS", "statusCode": "00", "amount": "10000", "currency": "TZS", "msisdn": "255712345678", "timestamp": "2024-01-15T10:30:00Z", "type": "Collection"},
						{"id": "txn-002", "reference": "ref-002", "status": "FAILED", "statusCode": "02", "amount": "50000", "currency": "TZS", "msisdn": "255712345678", "timestamp": "2024-01-15T11:00:00Z", "type": "Disbursement"}
					],
					"summary": {"totalCount": 2, "totalAmount": "60000", "currency": "TZS", "successfulCount": 1, "failedCount": 1, "pendingCount": 0}
				}
			}`,
			wantCode:    "200",
			wantTxnCnt:  2,
			wantSummary: true,
		},
		{
			name: "without summary (omitempty)",
			jsonInput: `{
				"status": {"code": "200", "message": "success", "success": true},
				"data": {
					"transactions": [
						{"id": "txn-001", "status": "SUCCESS", "amount": "10000", "currency": "TZS"}
					]
				}
			}`,
			wantCode:    "200",
			wantTxnCnt:  1,
			wantSummary: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp ReconciliationResponse
			if err := json.Unmarshal([]byte(tt.jsonInput), &resp); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if resp.Status.Code != tt.wantCode {
				t.Errorf("status code: got %s, want %s", resp.Status.Code, tt.wantCode)
			}
			if len(resp.Data.Transactions) != tt.wantTxnCnt {
				t.Fatalf("transaction count: got %d, want %d", len(resp.Data.Transactions), tt.wantTxnCnt)
			}

			if tt.wantSummary {
				if resp.Data.Summary == nil {
					t.Fatal("expected summary, got nil")
				}
				if resp.Data.Summary.TotalCount != tt.wantTxnCnt {
					t.Errorf("summary totalCount: got %d, want %d", resp.Data.Summary.TotalCount, tt.wantTxnCnt)
				}
			} else {
				if resp.Data.Summary != nil {
					t.Errorf("expected no summary, got: %+v", resp.Data.Summary)
				}
			}

			// Marshal back and compare
			data, err := json.Marshal(resp)
			if err != nil {
				t.Fatalf("failed to marshal back: %v", err)
			}
			if !jsonEqualString(tt.jsonInput, string(data)) {
				t.Errorf("roundtrip JSON differs\noriginal: %s\nroundtrip: %s", tt.jsonInput, string(data))
			}
		})
	}
}

func TestErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		jsonInput  string
		targetType string
	}{
		{
			name:       "malformed JSON",
			jsonInput:  `{invalid json}`,
			targetType: "CollectionResponse",
		},
		{
			name:       "status.success as string instead of bool",
			jsonInput:  `{"status": {"code": "200", "message": "success", "success": "true"}, "data": {"transaction": {"id": "txn-001", "status": "PENDING"}}}`,
			targetType: "CollectionResponse",
		},
		{
			name:       "amount as number instead of string",
			jsonInput:  `{"reference": "ref", "subscriber": {"country": "TZ", "currency": "TZS", "msisdn": "255712345678"}, "transaction": {"amount": 10000, "country": "TZ", "currency": "TZS", "id": "txn-001", "type": "Collection"}}`,
			targetType: "CollectionRequest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch tt.targetType {
			case "CollectionResponse":
				var resp CollectionResponse
				if err := json.Unmarshal([]byte(tt.jsonInput), &resp); err == nil {
					t.Errorf("expected error for malformed input, got nil")
				}
			case "CollectionRequest":
				var req CollectionRequest
				if err := json.Unmarshal([]byte(tt.jsonInput), &req); err == nil {
					t.Errorf("expected error for malformed input, got nil")
				}
			}
		})
	}
}

// TransactionStatusRequest has no JSON body (transaction ID is a URL path parameter),
// so no marshalling tests are needed for it.

// contains checks if a substring exists in a string.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || indexSubstring(s, substr) >= 0))
}

func indexSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// jsonEqualString compares two JSON strings for semantic equality (key-order independent).
func jsonEqualString(a, b string) bool {
	var va, vb map[string]any
	if err := json.Unmarshal([]byte(a), &va); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(b), &vb); err != nil {
		return false
	}
	return jsonEqual(va, vb)
}

// jsonEqual compares two map[string]any for semantic equality.
func jsonEqual(a, b map[string]any) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		bv, ok := b[k]
		if !ok {
			return false
		}
		if !jsonEqualValue(v, bv) {
			return false
		}
	}
	return true
}

func jsonEqualValue(a, b any) bool {
	switch av := a.(type) {
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		return jsonEqual(av, bv)
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonEqualValue(av[i], bv[i]) {
				return false
			}
		}
		return true
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && av == bv
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case nil:
		return b == nil
	default:
		return false
	}
}
