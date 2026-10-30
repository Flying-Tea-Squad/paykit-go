package airtel

// CollectionRequest represents a payment collection request (USSD push).
// POST /merchant/v2/payments/
//
// The subscriber and transaction objects mirror the Airtel Money Open API structure.
// Amount is sent as a string in major units per the API specification.
// PIN is optional and depends on merchant configuration.
type CollectionRequest struct {
	Reference string `json:"reference"`
	Subscriber struct {
		Country  string `json:"country"`
		Currency string `json:"currency"`
		MSISDN   string `json:"msisdn"`
	} `json:"subscriber"`
	Transaction struct {
		Amount   string `json:"amount"`
		Country  string `json:"country"`
		Currency string `json:"currency"`
		ID       string `json:"id"`
		Type     string `json:"type"`
	} `json:"transaction"`
	PIN string `json:"pin,omitempty"`
}

// CollectionResponse represents the response from a payment collection request.
// The response envelope contains data.transaction with status details.
type CollectionResponse struct {
	Data struct {
		Transaction struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Message string `json:"message,omitempty"`
			// StatusCode is the provider-specific status code (e.g., "00", "02").
			StatusCode string `json:"statusCode,omitempty"`
		} `json:"transaction"`
	} `json:"data"`
}

// DisbursementRequest represents a disbursement/payout request.
// POST /standard/v2/disbursements/
//
// NOTE: The official Airtel Money developer portal documents this as
// POST /merchant/v2/payouts/ with a "payee" object instead of "subscriber".
// This struct follows the issue specification using /standard/v2/disbursements/
// and the subscriber/transaction nesting. Adjust if the provider confirms otherwise.
type DisbursementRequest struct {
	Reference string `json:"reference"`
	Subscriber struct {
		Country  string `json:"country"`
		Currency string `json:"currency"`
		MSISDN   string `json:"msisdn"`
	} `json:"subscriber"`
	Transaction struct {
		Amount   string `json:"amount"`
		Country  string `json:"country"`
		Currency string `json:"currency"`
		ID       string `json:"id"`
		Type     string `json:"type"`
	} `json:"transaction"`
}

// DisbursementResponse represents the response from a disbursement request.
type DisbursementResponse struct {
	Data struct {
		Transaction struct {
			ID           string `json:"id"`
			Status       string `json:"status"`
			StatusCode   string `json:"statusCode,omitempty"`
			Message      string `json:"message,omitempty"`
			// TransactionID is the provider's unique transaction reference for the payout.
			TransactionID string `json:"transactionId,omitempty"`
		} `json:"transaction"`
	} `json:"data"`
}

// TransactionStatusRequest represents a transaction status query request.
// GET /standard/v1/payments/{id}
//
// The transaction ID is carried as a URL path parameter; this struct has no JSON body.
// It exists to provide a typed request for the gateway implementation.
type TransactionStatusRequest struct {
	TransactionID string
}

// TransactionStatusResponse represents the response from a transaction status query.
type TransactionStatusResponse struct {
	Data struct {
		Transaction struct {
			ID        string `json:"id"`
			Reference string `json:"reference,omitempty"`
			Status    string `json:"status"`
			StatusCode string `json:"statusCode,omitempty"`
			Amount    string `json:"amount,omitempty"`
			Currency  string `json:"currency,omitempty"`
			MSISDN    string `json:"msisdn,omitempty"`
			Timestamp string `json:"timestamp,omitempty"`
		} `json:"transaction"`
	} `json:"data"`
}

// ReconciliationRequest represents a reconciliation query request.
//
// NOTE: The Airtel Money Open API does not expose a single, well-documented
// "reconciliation" endpoint in the public developer portal. This type is a
// best-effort design based on common reconciliation patterns (date range + optional
// transaction reference filter). The actual endpoint and payload may differ;
// adjust when the provider publishes definitive docs.
type ReconciliationRequest struct {
	// FromDate is the start of the reconciliation window (RFC3339 date).
	FromDate string `json:"fromDate"`
	// ToDate is the end of the reconciliation window (RFC3339 date).
	ToDate string `json:"toDate"`
	// Reference optionally filters to a specific merchant reference.
	Reference string `json:"reference,omitempty"`
	// TransactionID optionally filters to a specific provider transaction ID.
	TransactionID string `json:"transactionId,omitempty"`
}

// ReconciliationResponse represents the response from a reconciliation query.
//
// NOTE: As with ReconciliationRequest, the response shape is inferred from
// common patterns. The provider's actual response may include different fields
// or nesting. Update when definitive documentation is available.
type ReconciliationResponse struct {
	Data struct {
		Transactions []struct {
			ID            string `json:"id"`
			Reference     string `json:"reference,omitempty"`
			Status        string `json:"status"`
			StatusCode    string `json:"statusCode,omitempty"`
			Amount        string `json:"amount,omitempty"`
			Currency      string `json:"currency,omitempty"`
			MSISDN        string `json:"msisdn,omitempty"`
			Timestamp     string `json:"timestamp,omitempty"`
			Type          string `json:"type,omitempty"` // e.g., "Collection", "Disbursement"
		} `json:"transactions"`
		// Summary provides aggregate totals for the reconciliation period.
		Summary struct {
			TotalCount     int    `json:"totalCount,omitempty"`
			TotalAmount    string `json:"totalAmount,omitempty"`
			Currency       string `json:"currency,omitempty"`
			SuccessfulCount int   `json:"successfulCount,omitempty"`
			FailedCount    int    `json:"failedCount,omitempty"`
			PendingCount   int    `json:"pendingCount,omitempty"`
		} `json:"summary,omitempty"`
	} `json:"data"`
}
