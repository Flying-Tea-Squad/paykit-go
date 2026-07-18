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