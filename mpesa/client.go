package mpesa

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

type Client struct {
	baseURL string
	httpClient *http.Client

}

func (c *Client) STKPush(ctx context.Context, req STKPushRequest) (*STKPushRequest, error){

	body, err := json.Marshal(req)

	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, 
		http.MethodPost,
		c.baseURL+"/mpesa/stkpush/v1/processrequest",
		bytes.NewReader(body),
	)

	if err != nil {
		return nil,err
	}

	_ = httpReq

	return &STKPushResponse{}, nil
	// req, err := http.NewRequest("POST", "/mpesa/stkpush/v1/processrequest", )
}