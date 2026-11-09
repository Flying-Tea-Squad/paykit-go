package airtel

// StatusEnvelope represents the top-level status object returned by Airtel Money API responses.
type StatusEnvelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// Subscriber represents a subscriber (payer) in a collection request.
type Subscriber struct {
	Country  string `json:"country"`
	Currency string `json:"currency"`
	MSISDN   string `json:"msisdn"`
}

// CollectionTransaction represents the transaction details in a collection request.
type CollectionTransaction struct {
	Amount   string `json:"amount"`
	Country  string `json:"country"`
	Currency string `json:"currency"`
	ID       string `json:"id"`
	Type     string `json:"type"`
}

// CollectionRequest represents a payment collection request (USSD push).
// POST /merchant/v2/payments/
//
// The subscriber and transaction objects mirror the Airtel Money Open API structure.
// Amount is sent as a string in major units per the API specification.
type CollectionRequest struct {
	Reference   string                `json:"reference"`
	Subscriber  Subscriber            `json:"subscriber"`
	Transaction CollectionTransaction `json:"transaction"`
}

// CollectionResponse represents the response from a payment collection request.
type CollectionResponse struct {
	Status StatusEnvelope `json:"status"`
	Data   struct {
		Transaction struct {
			ID         string `json:"id"`
			Status     string `json:"status"`
			Message    string `json:"message,omitempty"`
			StatusCode string `json:"statusCode,omitempty"`
		} `json:"transaction"`
	} `json:"data"`
}

// Payee represents a payee (recipient) in a disbursement request.
type Payee struct {
	Country   string `json:"country"`
	Currency  string `json:"currency"`
	MSISDN    string `json:"msisdn"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// DisbursementTransaction represents the transaction details in a disbursement request.
type DisbursementTransaction struct {
	Amount   string `json:"amount"`
	Country  string `json:"country"`
	Currency string `json:"currency"`
	ID       string `json:"id"`
	Type     string `json:"type"`
}

// DisbursementRequest represents a disbursement/payout request.
// POST /standard/v2/disbursements/
//
// The official Airtel Money developer portal documents this as
// POST /standard/v1/disbursements/ (or v2) with a "payee" object and merchant "pin".
type DisbursementRequest struct {
	Reference   string                  `json:"reference"`
	PIN         string                  `json:"pin"`
	Payee       Payee                   `json:"payee"`
	Transaction DisbursementTransaction `json:"transaction"`
}

// DisbursementResponse represents the response from a disbursement request.
type DisbursementResponse struct {
	Status StatusEnvelope `json:"status"`
	Data   struct {
		Transaction struct {
			ID            string `json:"id"`
			Status        string `json:"status"`
			StatusCode    string `json:"statusCode,omitempty"`
			Message       string `json:"message,omitempty"`
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

// TransactionStatusData represents the data portion of a transaction status response.
type TransactionStatusData struct {
	Transaction struct {
		AirtelMoneyID string `json:"airtel_money_id,omitempty"`
		ID            string `json:"id"`
		Message       string `json:"message,omitempty"`
		Status        string `json:"status"`
		StatusCode    string `json:"statusCode,omitempty"`
	} `json:"transaction"`
}

// TransactionStatusResponse represents the response from a transaction status query.
type TransactionStatusResponse struct {
	Status StatusEnvelope        `json:"status"`
	Data   TransactionStatusData `json:"data"`
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
	Status StatusEnvelope `json:"status"`
	Data   struct {
		Transactions []struct {
			ID         string `json:"id"`
			Reference  string `json:"reference,omitempty"`
			Status     string `json:"status"`
			StatusCode string `json:"statusCode,omitempty"`
			Amount     string `json:"amount,omitempty"`
			Currency   string `json:"currency,omitempty"`
			MSISDN     string `json:"msisdn,omitempty"`
			Timestamp  string `json:"timestamp,omitempty"`
			Type       string `json:"type,omitempty"` // e.g., "Collection", "Disbursement"
		} `json:"transactions"`
		// Summary provides aggregate totals for the reconciliation period.
		Summary struct {
			TotalCount      int    `json:"totalCount,omitempty"`
			TotalAmount     string `json:"totalAmount,omitempty"`
			Currency        string `json:"currency,omitempty"`
			SuccessfulCount int    `json:"successfulCount,omitempty"`
			FailedCount     int    `json:"failedCount,omitempty"`
			PendingCount    int    `json:"pendingCount,omitempty"`
		} `json:"summary,omitempty"`
	} `json:"data"`
}
