package mpesa

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

type STKPushResponse struct {
	MerchantRequestID string `json:"MerchantRequestID"`
	CheckoutRequestID string `json:"CheckoutRequestID"`
	ResponseCode      string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
	CustomerMessage   string `json:"CustomerMessage"`
}

type C2BRegisterRequest struct {
	IdempotencyKey string `json:"-"`
	ShortCode string `json:"ShortCode"`
	ResponseType string `json:"ResponseType"`
	ConfirmationURL string `json:"ConfirmationURL"`
	ValidationURL string `json:"ValidationURL"`
}

type CB2SimulateRequest struct {
	IdempotencyKey string `json:"-"`
	ShortCode string `json:"ShortCode"`
	CommandID string `json:"CommandID"`
	Amount int `json:"Amount"`
	Msisdn string `json:"Msisdn"`
	BillRefNumber string `json:"BillRefNumber"`
}

type B2CRequest struct {
	IdempotencyKey string `json:"-"`
	InitiatorName string `json:"InitiatorName"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID string `json:"CommandID"`
	Amount int `json:"Amount"`
	PartyA string `json:"PartyA"`
	PartyB string `json:"PartyB"`
	Remarks string `json:"Remarks"`
	QueueTimeOutURL string `json:"QueueTimeOutURL"`
	ResultURL string `json:"ResultURL"`
	Occasion string `json:"Occasion,omitempty"`
}

type B2CResponse struct {
	ConversationID string `json:"ConversationID"`
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ResponseCode string `json:"ResponseCode"`
	ResponseDescription string `json:"ResponseDescription"`
}

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

type StatusResponse struct {
	OriginatorConversationID string `json:"OriginatorConversationID"`
	ConversationID           string `json:"ConversationID"`
	ResponseCode             string `json:"ResponseCode"`
	ResponseDescription      string `json:"ResponseDescription"`
}

// BalanceRequest initiates a real-time query of your M-PESA account balances.
type BalanceRequest struct {
	IdempotencyKey     string `json:"-"`
	Initiator          string `json:"Initiator"`
	SecurityCredential string `json:"SecurityCredential"`
	CommandID          string `json:"CommandID"` // Must be set to "AccountBalance"
	PartyA             string `json:"PartyA"`    // The short code of the organization
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

// QRRequest encapsulates the configuration for creating an M-PESA dynamic QR code.
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