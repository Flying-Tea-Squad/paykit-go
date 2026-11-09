package mpesa

// STKPushRequest encapsulates parameters required to initiate an STK Push (Lipa Na M-Pesa Online) customer charge.
type STKPushRequest struct {
	IdempotencyKey    string `json:"-"`
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	TransactionType   string `json:"TransactionType"`
	Amount            int    `json:"Amount"`
	PartyA            string `json:"PartyA"`
	PartyB            string `json:"PartyB"`
	PhoneNumber       string `json:"PhoneNumber"`
	CallBackURL       string `json:"CallBackURL"`
	AccountReference  string `json:"AccountReference"`
	TransactionDesc   string `json:"TransactionDesc"`
}

// STKPushResponse represents Daraja's immediate synchronous response to an STK Push initiation.
type STKPushResponse struct {
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage     string `json:"CustomerMessage"`
}

// STKPushQueryRequest encapsulates parameters required to query the status of an STK Push transaction.
type STKPushQueryRequest struct {
	IdempotencyKey    string `json:"-"`
	BusinessShortCode string `json:"BusinessShortCode"`
	Password          string `json:"Password"`
	Timestamp         string `json:"Timestamp"`
	CheckoutRequestID string `json:"CheckoutRequestID"`
}

// STKPushQueryResponse represents Daraja's response to an STK Push query request.
type STKPushQueryResponse struct {
	ResponseCode        string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	MerchantRequestID   string `json:"MerchantRequestID"`
	CheckoutRequestID   string `json:"CheckoutRequestID"`
	ResultCode          string `json:"ResultCode"`
	ResultDesc          string `json:"ResultDesc"`
}

// C2BRegisterRequest encapsulates configuration parameters for registering C2B validation and confirmation URLs.
type C2BRegisterRequest struct {
	IdempotencyKey  string `json:"-"`
	ShortCode       string `json:"ShortCode"`
	ResponseType    string `json:"ResponseType"`
	ConfirmationURL string `json:"ConfirmationURL"`
	ValidationURL   string `json:"ValidationURL"`
}

// CB2SimulateRequest encapsulates parameters to simulate a C2B transaction in sandbox environments.
type CB2SimulateRequest struct {
	IdempotencyKey string `json:"-"`
	ShortCode      string `json:"ShortCode"`
	CommandID      string `json:"CommandID"`
	Amount         int    `json:"Amount"`
	Msisdn         string `json:"Msisdn"`
	BillRefNumber  string `json:"BillRefNumber"`
}

// B2CRequest encapsulates parameters required to initiate a Business-to-Customer (B2C) payout.
type B2CRequest struct {
	IdempotencyKey     string `json:"-"`
	InitiatorName      string `json:"InitiatorName"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`
	Amount             int    `json:"Amount"`
	PartyA             string `json:"PartyA"`
	PartyB             string `json:"PartyB"`
	Remarks            string `json:"Remarks"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	ResultURL          string `json:"ResultURL"`
	Occasion           string `json:"Occasion,omitempty"`
}

// B2CResponse represents Daraja's immediate acknowledgment of a B2C payment request.
type B2CResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// StatusRequest encapsulates parameters required to query the status of an M-Pesa transaction via the transaction status endpoint.
type StatusRequest struct {
	IdempotencyKey         string `json:"-"`
	Initiator              string `json:"Initiator"`
	SecurityCredential     string `json:"SecurityCredential"`
	CommandID              string `json:"CommandID"`
	TransactionID          string `json:"TransactionID"`
	OriginalConversationID string `json:"OriginalConversationID"` // Note: The schema explicitly uses 'Original' here
	PartyA                 string `json:"PartyA"`
	IdentifierType         string `json:"IdentifierType"`
	ResultURL              string `json:"ResultURL"`
	QueueTimeOutURL        string `json:"QueueTimeOutURL"`
	Remarks                string `json:"Remarks"`
	Occasion               string `json:"Occasion,omitempty"`
}

// StatusResponse represents Daraja's immediate acknowledgment of a transaction status query request.
type StatusResponse struct {
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ConversationID           string `json:"ConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// ReversalRequest encapsulates parameters required to initiate an M-Pesa transaction reversal.
type ReversalRequest struct {
	IdempotencyKey         string `json:"-"`
	Initiator              string `json:"Initiator"`
	SecurityCredential     string `json:"SecurityCredential"`
	CommandID              string `json:"CommandID"`
	TransactionID          string `json:"TransactionID"`
	Amount                 int    `json:"Amount"`
	ReceiverParty          string `json:"ReceiverParty"`
	RecieverIdentifierType string `json:"RecieverIdentifierType"`
	ResultURL              string `json:"ResultURL"`
	QueueTimeOutURL        string `json:"QueueTimeOutURL"`
	Remarks                string `json:"Remarks"`
	Occasion               string `json:"Occasion,omitempty"`
}

// ReversalResponse represents Daraja's immediate acknowledgment of a reversal request.
type ReversalResponse struct {
	ConversationID           string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// BalanceRequest initiates a real-time query of your M-PESA account balances.
type BalanceRequest struct {
	IdempotencyKey     string `json:"-"`
	Initiator          string `json:"Initiator"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"`      // Must be set to "AccountBalance"
	PartyA             string `json:"PartyA"`         // The short code of the organization
	IdentifierType     string `json:"IdentifierType"` // Typically "4" for short codes
	Remarks            string `json:"Remarks"`
	QueueTimeOutURL    string `json:"QueueTimeOutURL"`
	ResultURL          string `json:"ResultURL"`
}

// BalanceResponse represents Daraja's immediate synchronous acknowledgment.
type BalanceResponse struct {
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ConversationID           string `json:"ConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// DynamicQRRequest encapsulates the configuration for creating an M-PESA dynamic QR code.
type DynamicQRRequest struct {
	IdempotencyKey string `json:"-"`
	MerchantName   string `json:"MerchantName"`
	RefNo          string `json:"RefNo"`
	Amount         uint32 `json:"Amount"`
	TrxCode        string `json:"TrxCode"` // Valid values: BG, WA, PB, SM, SB
	CPI            string `json:"CPI"`     // Credit Party Identifier (Shortcode, MSISDN, Till)
	Size           string `json:"Size"`    // Image width/height in pixels
}

// QRResponse delivers the synchronous payload containing the base64 image data.
type DynamicQRResponse struct {
	ResponseCode        string `json:"ResponseCode"`
	RequestID           string `json:"RequestID"`
	ResponseDescription string `json:"ResponseDescription"`
	QRCode              string `json:"QRCode"` // Base64 encoded string representing the square QR image
}
