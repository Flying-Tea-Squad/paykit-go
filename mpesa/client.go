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

func (c *Client) STKPush(ctx context.Context, req STKPushRequest) (*STKPushResponse, error){

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

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stkResp STKPushResponse

	err = json.NewDecoder(resp.Body).Decode(&stkResp)
	if err != nil {
		return nil, err
	}

	return &stkResp, nil
	// req, err := http.NewRequest("POST", "/mpesa/stkpush/v1/processrequest", )
}