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
// POST /merchant/v1/payments/
//
// The subscriber and transaction objects mirror the Airtel Money Open API structure.
// Amount is sent as a string in major units per the API specification.
type CollectionRequest struct {
	Reference   string                `json:"reference"`
	Subscriber  Subscriber            `json:"subscriber"`
	Transaction CollectionTransaction `json:"transaction"`
}

// CollectionResponseData represents the data portion of a collection response.
type CollectionResponseData struct {
	Transaction CollectionResponseTransaction `json:"transaction"`
}

// CollectionResponseTransaction represents the transaction in a collection response.
type CollectionResponseTransaction struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Message    string `json:"message,omitempty"`
	StatusCode string `json:"statusCode,omitempty"`
}

// CollectionResponse represents the response from a payment collection request.
type CollectionResponse struct {
	Status StatusEnvelope         `json:"status"`
	Data   CollectionResponseData `json:"data"`
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
// POST /standard/v1/disbursements/ (v2 also exists)
//
// The request uses a "payee" object and requires the merchant "pin".
type DisbursementRequest struct {
	Reference   string                  `json:"reference"`
	PIN         string                  `json:"pin"`
	Payee       Payee                   `json:"payee"`
	Transaction DisbursementTransaction `json:"transaction"`
}

// DisbursementResponseData represents the data portion of a disbursement response.
type DisbursementResponseData struct {
	Transaction DisbursementResponseTransaction `json:"transaction"`
}

// DisbursementResponseTransaction represents the transaction in a disbursement response.
type DisbursementResponseTransaction struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	StatusCode    string `json:"statusCode,omitempty"`
	Message       string `json:"message,omitempty"`
	TransactionID string `json:"transactionId,omitempty"`
}

// DisbursementResponse represents the response from a disbursement request.
type DisbursementResponse struct {
	Status StatusEnvelope           `json:"status"`
	Data   DisbursementResponseData `json:"data"`
}

// TransactionStatusRequest represents a transaction status query request.
// GET /standard/v1/payments/{id}
//
// The transaction ID is carried as a URL path parameter; this struct has no JSON body.
// It exists to provide a typed request for the gateway implementation.
type TransactionStatusRequest struct {
	TransactionID string
}

// TransactionStatusTransaction represents the transaction in a transaction status response.
type TransactionStatusTransaction struct {
	AirtelMoneyID string `json:"airtel_money_id,omitempty"`
	ID            string `json:"id"`
	Message       string `json:"message,omitempty"`
	Status        string `json:"status"`
	StatusCode    string `json:"statusCode,omitempty"`
}

// TransactionStatusData represents the data portion of a transaction status response.
type TransactionStatusData struct {
	Transaction TransactionStatusTransaction `json:"transaction"`
}

// TransactionStatusResponse represents the response from a transaction status query.
type TransactionStatusResponse struct {
	Status StatusEnvelope        `json:"status"`
	Data   TransactionStatusData `json:"data"`
}

// ReconciliationRequest represents a reconciliation query request.
//
// NOTE: Airtel's public Open API has no standalone reconciliation endpoint;
// reconciliation is done via transaction status queries or settlement reports.
// These types are provisional and may be refined or deprecated in later milestones.
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

// ReconciliationTransaction represents a single transaction in a reconciliation response.
type ReconciliationTransaction struct {
	ID         string `json:"id"`
	Reference  string `json:"reference,omitempty"`
	Status     string `json:"status"`
	StatusCode string `json:"statusCode,omitempty"`
	Amount     string `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
	MSISDN     string `json:"msisdn,omitempty"`
	Timestamp  string `json:"timestamp,omitempty"`
	Type       string `json:"type,omitempty"` // e.g., "Collection", "Disbursement"
}

// ReconciliationSummary provides aggregate totals for the reconciliation period.
// Count fields omit omitempty to preserve zero values (e.g., pendingCount: 0).
type ReconciliationSummary struct {
	TotalCount      int    `json:"totalCount"`
	TotalAmount     string `json:"totalAmount,omitempty"`
	Currency        string `json:"currency,omitempty"`
	SuccessfulCount int    `json:"successfulCount"`
	FailedCount     int    `json:"failedCount"`
	PendingCount    int    `json:"pendingCount"`
}

// ReconciliationResponseData represents the data portion of a reconciliation response.
type ReconciliationResponseData struct {
	Transactions []ReconciliationTransaction `json:"transactions"`
	Summary      *ReconciliationSummary      `json:"summary,omitempty"`
}

// ReconciliationResponse represents the response from a reconciliation query.
//
// NOTE: Airtel's public Open API has no standalone reconciliation endpoint;
// reconciliation is done via transaction status queries or settlement reports.
// These types are provisional and may be refined or deprecated in later milestones.
type ReconciliationResponse struct {
	Status StatusEnvelope             `json:"status"`
	Data   ReconciliationResponseData `json:"data"`
}
