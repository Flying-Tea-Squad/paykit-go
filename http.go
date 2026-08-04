package paykit

import "net/http"

// HTTPClient executes HTTP requests for payment provider clients.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
