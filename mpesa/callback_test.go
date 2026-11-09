package mpesa

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseSTKPushCallback(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    *STKPushCallbackResult
	}{
		{
			name: "successful callback",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 0,
						"ResultDesc": "The service request is processed successfully.",
						"CallbackMetadata": {
							"Item": [
								{"Name": "Amount", "Value": 500},
								{"Name": "MpesaReceiptNumber", "Value": "ABC123XYZ"},
								{"Name": "TransactionDate", "Value": 20260802140530},
								{"Name": "PhoneNumber", "Value": 254700000000}
							]
						}
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        0,
				ResultDesc:        "The service request is processed successfully.",
				Msisdn:            "254700000000",
				Amount:            500,
				TransactionID:     "ABC123XYZ",
				ReferenceData: map[string]json.RawMessage{
					"Amount":             json.RawMessage("500"),
					"MpesaReceiptNumber": json.RawMessage(`"ABC123XYZ"`),
					"TransactionDate":    json.RawMessage("20260802140530"),
					"PhoneNumber":        json.RawMessage("254700000000"),
				},
			},
		},
		{
			name: "unsuccessful callback without metadata",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 1032,
						"ResultDesc": "Request cancelled by user"
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        1032,
				ResultDesc:        "Request cancelled by user",
			},
		},
		{
			name: "metadata order does not affect extraction",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 0,
						"ResultDesc": "Success",
						"CallbackMetadata": {
							"Item": [
								{"Name": "PhoneNumber", "Value": 254711111111},
								{"Name": "MpesaReceiptNumber", "Value": "ORDER123"},
								{"Name": "Amount", "Value": 75}
							]
						}
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        0,
				ResultDesc:        "Success",
				Msisdn:            "254711111111",
				Amount:            75,
				TransactionID:     "ORDER123",
				ReferenceData: map[string]json.RawMessage{
					"PhoneNumber":        json.RawMessage("254711111111"),
					"MpesaReceiptNumber": json.RawMessage(`"ORDER123"`),
					"Amount":             json.RawMessage("75"),
				},
			},
		},
		{
			name: "unknown metadata is preserved",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 0,
						"ResultDesc": "Success",
						"CallbackMetadata": {
							"Item": [
								{"Name": "Amount", "Value": 120},
								{"Name": "MpesaReceiptNumber", "Value": "UNKNOWN1"},
								{"Name": "PhoneNumber", "Value": 254722222222},
								{"Name": "FutureField", "Value": "future-value"}
							]
						}
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        0,
				ResultDesc:        "Success",
				Msisdn:            "254722222222",
				Amount:            120,
				TransactionID:     "UNKNOWN1",
				ReferenceData: map[string]json.RawMessage{
					"Amount":             json.RawMessage("120"),
					"MpesaReceiptNumber": json.RawMessage(`"UNKNOWN1"`),
					"PhoneNumber":        json.RawMessage("254722222222"),
					"FutureField":        json.RawMessage(`"future-value"`),
				},
			},
		},
		{
			name: "quoted phone number is normalized",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 0,
						"ResultDesc": "Success",
						"CallbackMetadata": {
							"Item": [
								{"Name": "Amount", "Value": 10},
								{"Name": "MpesaReceiptNumber", "Value": "STRING1"},
								{"Name": "PhoneNumber", "Value": "254733333333"}
							]
						}
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        0,
				ResultDesc:        "Success",
				Msisdn:            "254733333333",
				Amount:            10,
				TransactionID:     "STRING1",
				ReferenceData: map[string]json.RawMessage{
					"Amount":             json.RawMessage("10"),
					"MpesaReceiptNumber": json.RawMessage(`"STRING1"`),
					"PhoneNumber":        json.RawMessage(`"254733333333"`),
				},
			},
		},
		{
			name: "decimal amount with trailing zeros",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 0,
						"ResultDesc": "Success",
						"CallbackMetadata": {
							"Item": [
								{"Name": "Amount", "Value": 1.00},
								{"Name": "MpesaReceiptNumber", "Value": "DECIMAL1"},
								{"Name": "PhoneNumber", "Value": 254700000000}
							]
						}
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        0,
				ResultDesc:        "Success",
				Msisdn:            "254700000000",
				Amount:            1,
				TransactionID:     "DECIMAL1",
				ReferenceData: map[string]json.RawMessage{
					"Amount":             json.RawMessage("1.00"),
					"MpesaReceiptNumber": json.RawMessage(`"DECIMAL1"`),
					"PhoneNumber":        json.RawMessage("254700000000"),
				},
			},
		},
		{
			name: "empty optional metadata item",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 0,
						"ResultDesc": "Success",
						"CallbackMetadata": {
							"Item": [
								{"Name": "Amount", "Value": 250},
								{"Name": "MpesaReceiptNumber", "Value": "EMPTYBAL1"},
								{"Name": "PhoneNumber", "Value": 254700000000},
								{"Name": "Balance"}
							]
						}
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        0,
				ResultDesc:        "Success",
				Msisdn:            "254700000000",
				Amount:            250,
				TransactionID:     "EMPTYBAL1",
				ReferenceData: map[string]json.RawMessage{
					"Amount":             json.RawMessage("250"),
					"MpesaReceiptNumber": json.RawMessage(`"EMPTYBAL1"`),
					"PhoneNumber":        json.RawMessage("254700000000"),
				},
			},
		},
		{
			name: "null optional metadata item",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 0,
						"ResultDesc": "Success",
						"CallbackMetadata": {
							"Item": [
								{"Name": "Amount", "Value": 250},
								{"Name": "MpesaReceiptNumber", "Value": "NULLBAL1"},
								{"Name": "PhoneNumber", "Value": 254700000000},
								{"Name": "Balance", "Value": null}
							]
						}
					}
				}
			}`,
			want: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        0,
				ResultDesc:        "Success",
				Msisdn:            "254700000000",
				Amount:            250,
				TransactionID:     "NULLBAL1",
				ReferenceData: map[string]json.RawMessage{
					"Amount":             json.RawMessage("250"),
					"MpesaReceiptNumber": json.RawMessage(`"NULLBAL1"`),
					"PhoneNumber":        json.RawMessage("254700000000"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSTKPushCallback([]byte(tt.payload))
			if err != nil {
				t.Fatalf("ParseSTKPushCallback() error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSTKPushCallback() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseSTKPushCallbackEnvelopeValidation(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr string
	}{
		{
			name:    "empty payload",
			payload: "",
			wantErr: "empty payload",
		},
		{
			name:    "whitespace-only payload",
			payload: " \n\t ",
			wantErr: "empty payload",
		},
		{
			name:    "malformed JSON",
			payload: `{"Body":`,
			wantErr: "invalid JSON",
		},
		{
			name: "multiple JSON values",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 1032,
						"ResultDesc": "Request cancelled by user"
					}
				}
			} {}`,
			wantErr: "multiple JSON values",
		},
		{
			name: "invalid trailing JSON",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 1032,
						"ResultDesc": "Request cancelled by user"
					}
				}
			} {`,
			wantErr: "invalid trailing JSON",
		},
		{
			name:    "missing callback object",
			payload: `{"Body": {}}`,
			wantErr: "missing Body.stkCallback",
		},
		{
			name:    "null callback object",
			payload: `{"Body": {"stkCallback": null}}`,
			wantErr: "missing Body.stkCallback",
		},
		{
			name: "missing merchant request ID",
			payload: `{
				"Body": {
					"stkCallback": {
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 1032,
						"ResultDesc": "Request cancelled by user"
					}
				}
			}`,
			wantErr: "missing MerchantRequestID",
		},
		{
			name: "missing checkout request ID",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"ResultCode": 1032,
						"ResultDesc": "Request cancelled by user"
					}
				}
			}`,
			wantErr: "missing CheckoutRequestID",
		},
		{
			name: "missing result code",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultDesc": "Request cancelled by user"
					}
				}
			}`,
			wantErr: "missing ResultCode",
		},
		{
			name: "missing result description",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 1032
					}
				}
			}`,
			wantErr: "missing ResultDesc",
		},
		{
			name: "whitespace-only merchant request ID",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "   ",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 1032,
						"ResultDesc": "Request cancelled by user"
					}
				}
			}`,
			wantErr: "missing MerchantRequestID",
		},
		{
			name: "whitespace-only checkout request ID",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "   ",
						"ResultCode": 1032,
						"ResultDesc": "Request cancelled by user"
					}
				}
			}`,
			wantErr: "missing CheckoutRequestID",
		},
		{
			name: "whitespace-only result description",
			payload: `{
				"Body": {
					"stkCallback": {
						"MerchantRequestID": "merchant-request-id",
						"CheckoutRequestID": "checkout-request-id",
						"ResultCode": 1032,
						"ResultDesc": "   "
					}
				}
			}`,
			wantErr: "missing ResultDesc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSTKPushCallback([]byte(tt.payload))
			if err == nil {
				t.Fatalf("ParseSTKPushCallback() result = %#v, want error containing %q", result, tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseSTKPushCallback() error = %q, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseSTKPushCallbackMetadataValidation(t *testing.T) {
	tests := []struct {
		name       string
		resultCode int
		metadata   string
		wantErr    string
	}{
		{
			name:       "successful callback without metadata",
			resultCode: 0,
			wantErr:    "successful callback missing CallbackMetadata",
		},
		{
			name:       "successful callback with empty metadata",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(""),
			wantErr:    `successful callback missing metadata item "Amount"`,
		},
		{
			name:       "successful callback missing amount",
			resultCode: 0,
			metadata: testSTKCallbackMetadata(`
				{"Name": "MpesaReceiptNumber", "Value": "RECEIPT1"},
				{"Name": "PhoneNumber", "Value": 254700000000}
			`),
			wantErr: `successful callback missing metadata item "Amount"`,
		},
		{
			name:       "successful callback missing receipt number",
			resultCode: 0,
			metadata: testSTKCallbackMetadata(`
				{"Name": "Amount", "Value": 100},
				{"Name": "PhoneNumber", "Value": 254700000000}
			`),
			wantErr: `successful callback missing metadata item "MpesaReceiptNumber"`,
		},
		{
			name:       "successful callback missing phone number",
			resultCode: 0,
			metadata: testSTKCallbackMetadata(`
				{"Name": "Amount", "Value": 100},
				{"Name": "MpesaReceiptNumber", "Value": "RECEIPT1"}
			`),
			wantErr: `successful callback missing metadata item "PhoneNumber"`,
		},
		{
			name:       "amount has string type",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": "100"}`),
			wantErr:    `metadata item "Amount" must be a whole number`,
		},
		{
			name:       "amount is zero",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": 0}`),
			wantErr:    `metadata item "Amount" must be greater than zero`,
		},
		{
			name:       "amount is zero decimal",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": 0.00}`),
			wantErr:    `metadata item "Amount" must be greater than zero`,
		},
		{
			name:       "amount is negative",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": -1}`),
			wantErr:    `metadata item "Amount" must be greater than zero`,
		},
		{
			name:       "amount is negative decimal",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": -1.00}`),
			wantErr:    `metadata item "Amount" must be greater than zero`,
		},
		{
			name:       "amount is fractional",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": 1.5}`),
			wantErr:    `metadata item "Amount" must be a whole number`,
		},
		{
			name:       "amount is fractional with trailing zeros",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": 1.50}`),
			wantErr:    `metadata item "Amount" must be a whole number`,
		},
		{
			name:       "amount has scientific notation",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": 1e2}`),
			wantErr:    `metadata item "Amount" must be a whole number`,
		},
		{
			name:       "receipt number has numeric type",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "MpesaReceiptNumber", "Value": 123}`),
			wantErr:    `metadata item "MpesaReceiptNumber" must be a string`,
		},
		{
			name:       "receipt number is empty",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "MpesaReceiptNumber", "Value": "  "}`),
			wantErr:    `metadata item "MpesaReceiptNumber" must not be empty`,
		},
		{
			name:       "phone number has boolean type",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "PhoneNumber", "Value": true}`),
			wantErr:    `metadata item "PhoneNumber" must be a number or string`,
		},
		{
			name:       "phone number is empty",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "PhoneNumber", "Value": " "}`),
			wantErr:    `metadata item "PhoneNumber" must not be empty`,
		},
		{
			name:       "phone number is zero",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "PhoneNumber", "Value": 0}`),
			wantErr:    `metadata item "PhoneNumber" must be a positive whole number or non-empty string`,
		},
		{
			name:       "phone number is fractional",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "PhoneNumber", "Value": 254700000000.5}`),
			wantErr:    `metadata item "PhoneNumber" must be a positive whole number or non-empty string`,
		},
		{
			name:       "metadata item has no name",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "", "Value": "value"}`),
			wantErr:    "metadata item has an empty Name",
		},
		{
			name:       "mandatory metadata item Amount has no value",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount"}`),
			wantErr:    `metadata item "Amount" is missing Value`,
		},
		{
			name:       "mandatory metadata item Amount has null value",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": null}`),
			wantErr:    `metadata item "Amount" is missing Value`,
		},
		{
			name:       "mandatory metadata item MpesaReceiptNumber has no value",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "MpesaReceiptNumber"}`),
			wantErr:    `metadata item "MpesaReceiptNumber" is missing Value`,
		},
		{
			name:       "mandatory metadata item MpesaReceiptNumber has null value",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "MpesaReceiptNumber", "Value": null}`),
			wantErr:    `metadata item "MpesaReceiptNumber" is missing Value`,
		},
		{
			name:       "mandatory metadata item PhoneNumber has no value",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "PhoneNumber"}`),
			wantErr:    `metadata item "PhoneNumber" is missing Value`,
		},
		{
			name:       "mandatory metadata item PhoneNumber has null value",
			resultCode: 0,
			metadata:   testSTKCallbackMetadata(`{"Name": "PhoneNumber", "Value": null}`),
			wantErr:    `metadata item "PhoneNumber" is missing Value`,
		},
		{
			name:       "metadata item name is duplicated",
			resultCode: 0,
			metadata: testSTKCallbackMetadata(`
				{"Name": "Amount", "Value": 100},
				{"Name": "Amount", "Value": 200}
			`),
			wantErr: `duplicate metadata item "Amount"`,
		},
		{
			name:       "known metadata remains validated on failed payment",
			resultCode: 1032,
			metadata:   testSTKCallbackMetadata(`{"Name": "Amount", "Value": "invalid"}`),
			wantErr:    `metadata item "Amount" must be a whole number`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSTKPushCallback([]byte(testSTKCallbackPayload(tt.resultCode, tt.metadata)))
			if err == nil {
				t.Fatalf("ParseSTKPushCallback() result = %#v, want error containing %q", result, tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("ParseSTKPushCallback() error = %q, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestParseSTKPushCallbackUnsuccessfulOutcomes(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		expected *STKPushCallbackResult
	}{
		{
			name:    "unknown result code without metadata",
			payload: testSTKCallbackPayload(9999, ""),
			expected: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        9999,
				ResultDesc:        "test result",
			},
		},
		{
			name: "failed payment with valid partial metadata",
			payload: testSTKCallbackPayload(1032, testSTKCallbackMetadata(`
				{"Name": "Amount", "Value": 25},
				{"Name": "FutureField", "Value": true}
			`)),
			expected: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        1032,
				ResultDesc:        "test result",
				Amount:            25,
				ReferenceData: map[string]json.RawMessage{
					"Amount":      json.RawMessage("25"),
					"FutureField": json.RawMessage("true"),
				},
			},
		},
		{
			name: "failed payment with empty balance item",
			payload: testSTKCallbackPayload(1032, testSTKCallbackMetadata(`
				{"Name": "Amount", "Value": 25},
				{"Name": "Balance"}
			`)),
			expected: &STKPushCallbackResult{
				MerchantRequestID: "merchant-request-id",
				CheckoutRequestID: "checkout-request-id",
				ResultCode:        1032,
				ResultDesc:        "test result",
				Amount:            25,
				ReferenceData: map[string]json.RawMessage{
					"Amount": json.RawMessage("25"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSTKPushCallback([]byte(tt.payload))
			if err != nil {
				t.Fatalf("ParseSTKPushCallback() error = %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseSTKPushCallback() = %#v, want %#v", result, tt.expected)
			}
		})
	}
}

func TestParseSTKPushCallbackAmount(t *testing.T) {
	tests := []struct {
		name    string
		data    json.RawMessage
		want    int
		wantErr string
	}{
		{
			name: "integral whole number",
			data: json.RawMessage("1"),
			want: 1,
		},
		{
			name: "integral decimal with single zero",
			data: json.RawMessage("1.0"),
			want: 1,
		},
		{
			name: "integral decimal with two zeros",
			data: json.RawMessage("1.00"),
			want: 1,
		},
		{
			name: "integral decimal with multiple zeros",
			data: json.RawMessage("500.0000"),
			want: 500,
		},
		{
			name: "large integral decimal",
			data: json.RawMessage("100000.00"),
			want: 100000,
		},
		{
			name:    "fractional amount",
			data:    json.RawMessage("1.50"),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "fractional amount without trailing zero",
			data:    json.RawMessage("1.5"),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "fractional amount with small fraction",
			data:    json.RawMessage("1.0001"),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "quoted string amount",
			data:    json.RawMessage(`"100"`),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "quoted decimal string amount",
			data:    json.RawMessage(`"1.00"`),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "boolean amount",
			data:    json.RawMessage("true"),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "null amount",
			data:    json.RawMessage("null"),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "empty raw message",
			data:    json.RawMessage(""),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "zero amount",
			data:    json.RawMessage("0"),
			wantErr: `metadata item "Amount" must be greater than zero`,
		},
		{
			name:    "zero decimal amount",
			data:    json.RawMessage("0.00"),
			wantErr: `metadata item "Amount" must be greater than zero`,
		},
		{
			name:    "negative amount",
			data:    json.RawMessage("-1"),
			wantErr: `metadata item "Amount" must be greater than zero`,
		},
		{
			name:    "negative decimal amount",
			data:    json.RawMessage("-1.00"),
			wantErr: `metadata item "Amount" must be greater than zero`,
		},
		{
			name:    "scientific notation",
			data:    json.RawMessage("1e2"),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
		{
			name:    "overflow integer",
			data:    json.RawMessage("999999999999999999999999999999999999999999999999999999999999"),
			wantErr: `metadata item "Amount" must be a whole number`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSTKPushCallbackAmount(tt.data)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseSTKPushCallbackAmount(%s) = %d, want error containing %q", string(tt.data), got, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("parseSTKPushCallbackAmount(%s) error = %q, want error containing %q", string(tt.data), err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseSTKPushCallbackAmount(%s) unexpected error: %v", string(tt.data), err)
			}
			if got != tt.want {
				t.Errorf("parseSTKPushCallbackAmount(%s) = %d, want %d", string(tt.data), got, tt.want)
			}
		})
	}
}

func testSTKCallbackPayload(resultCode int, metadata string) string {
	return fmt.Sprintf(`{
		"Body": {
			"stkCallback": {
				"MerchantRequestID": "merchant-request-id",
				"CheckoutRequestID": "checkout-request-id",
				"ResultCode": %d,
				"ResultDesc": "test result"%s
			}
		}
	}`, resultCode, metadata)
}

func testSTKCallbackMetadata(items string) string {
	return fmt.Sprintf(`,
		"CallbackMetadata": {
			"Item": [%s]
		}`, items)
}
