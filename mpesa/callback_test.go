package mpesa

import (
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
				ReferenceData: map[string]any{
					"Amount":             float64(500),
					"MpesaReceiptNumber": "ABC123XYZ",
					"TransactionDate":    float64(20260802140530),
					"PhoneNumber":        float64(254700000000),
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
				ReferenceData: map[string]any{
					"PhoneNumber":        float64(254711111111),
					"MpesaReceiptNumber": "ORDER123",
					"Amount":             float64(75),
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
				ReferenceData: map[string]any{
					"Amount":             float64(120),
					"MpesaReceiptNumber": "UNKNOWN1",
					"PhoneNumber":        float64(254722222222),
					"FutureField":        "future-value",
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
				ReferenceData: map[string]any{
					"Amount":             float64(10),
					"MpesaReceiptNumber": "STRING1",
					"PhoneNumber":        "254733333333",
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
			name:    "missing callback object",
			payload: `{"Body": {}}`,
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
