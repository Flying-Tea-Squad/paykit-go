package pesapal

// PesapalError represents an error returned by the Pesapal API.
type PesapalError struct {
	ErrorType   any    `json:"error_type"`
	Code        any    `json:"code"`
	Message     string `json:"message"`
	CallBackURL string `json:"call_back_url"`
}

// AuthRequest contains the credentials required to authenticate with Pesapal.
type AuthRequest struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
}

// AuthResponse represents the response returned by the Pesapal authentication endpoint.
type AuthResponse struct {
	Token      string        `json:"token"`
	ExpiryDate string        `json:"expiryDate"`
	Error      *PesapalError `json:"error"`
	Status     string        `json:"status"`
	Message    string        `json:"message"`
}

// IPNRegistrationRequest contains the details required to register an IPN URL.
type IPNRegistrationRequest struct {
	URL                 string `json:"url"`
	IPNNotificationType string `json:"ipn_notification_type"`
}

// IPNRegistrationResponse represents the response returned after registering an IPN URL.
type IPNRegistrationResponse struct {
	URL                            string        `json:"url"`
	CreatedDate                    string        `json:"created_date"`
	IPNID                          string        `json:"ipn_id"`
	NotificationType               int           `json:"notification_type"`
	IPNNotificationTypeDescription string        `json:"ipn_notification_type_description"`
	IPNStatus                      int           `json:"ipn_status"`
	IPNStatusDescription           string        `json:"ipn_status_description"`
	Error                          *PesapalError `json:"error"`
	Status                         string        `json:"status"`
}

// BillingAddress represents the billing details associated with a Pesapal order.
type BillingAddress struct {
	PhoneNumber  string `json:"phone_number,omitempty"`
	EmailAddress string `json:"email_address,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	MiddleName   string `json:"middle_name,omitempty"`
	LastName     string `json:"last_name,omitempty"`
	Line1        string `json:"line_1,omitempty"`
	Line2        string `json:"line_2,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	PostalCode   string `json:"postal_code,omitempty"`
	ZipCode      string `json:"zip_code,omitempty"`
}

// SubmitOrderRequest contains the details required to submit an order to Pesapal.
type SubmitOrderRequest struct {
	ID              string         `json:"id"`
	Currency        string         `json:"currency"`
	Amount          float64        `json:"amount"`
	Description     string         `json:"description"`
	CallbackURL     string         `json:"callback_url"`
	RedirectMode    string         `json:"redirect_mode,omitempty"`
	CancellationURL string         `json:"cancellation_url,omitempty"`
	NotificationID  string         `json:"notification_id"`
	Branch          string         `json:"branch,omitempty"`
	BillingAddress  BillingAddress `json:"billing_address"`
}

// SubmitOrderResponse represents the response returned after submitting an order.
type SubmitOrderResponse struct {
	OrderTrackingID   string        `json:"order_tracking_id"`
	MerchantReference string        `json:"merchant_reference"`
	RedirectURL       string        `json:"redirect_url"`
	Error             *PesapalError `json:"error"`
	Status            string        `json:"status"`
	Message           string        `json:"message"`
}

// TransactionStatusResponse represents the status and payment details of a Pesapal transaction.
type TransactionStatusResponse struct {
	PaymentMethod            string        `json:"payment_method"`
	Amount                   float64       `json:"amount"`
	CreatedDate              string        `json:"created_date"`
	ConfirmationCode         string        `json:"confirmation_code"`
	PaymentStatusDescription string        `json:"payment_status_description"`
	Description              string        `json:"description"`
	Message                  string        `json:"message"`
	PaymentAccount           string        `json:"payment_account"`
	CallBackURL              string        `json:"call_back_url"`
	StatusCode               int           `json:"status_code"`
	MerchantReference        string        `json:"merchant_reference"`
	PaymentStatusCode        string        `json:"payment_status_code"`
	Currency                 string        `json:"currency"`
	Error                    *PesapalError `json:"error"`
	Status                   string        `json:"status"`
}
