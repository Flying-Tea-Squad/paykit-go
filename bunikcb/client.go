package bunikcb

import (
	"net/http"
)

type Config struct {
	ConsumerKey string
	ConsumerSecret string
	BaseURL string
}

type Client struct {
	config Config
	httpClient *http.Client
}

// cfg is a parameter that takes type Config and returns a pointer to a Client AND and an error  
func New(cfg Config) (*Client, error) {
	// ensure the required values are present
	if cfg.ConsumerKey == "" {
		return nil, errors.New("consumer key is required")
	}

	if cfg.ConsumerSecret == "" {
		return nil, errors.New("consumer secret is required")
	}

	if cfg.BaseURL == "" {
		return nil, errors.New("baseURL is empty")
	}

	// http client that the Buni SDK will use(creating the HTTP client)
	httpClient :=&http.Client{}

	// initializes the fields of Client struct
	client :=&Client{
		config: cfg,
		httpClient: httpClient,
	}
	return client, nil
}