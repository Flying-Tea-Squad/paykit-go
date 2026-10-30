package pesapal

// IPNNotification is the payload Pesapal sends to the registered IPN URL.
type IPNNotification struct {
	OrderNotificationType  string `json:"OrderNotificationType"`
	OrderTrackingID        string `json:"OrderTrackingId"`
	OrderMerchantReference string `json:"OrderMerchantReference"`
}
