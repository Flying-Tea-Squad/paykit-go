package paykit

import (
	"net/http"
)

// HTTPClient defines the interface for executing HTTP requests.
// It is implemented by *http.Client and foundation HTTP client wrappers.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
