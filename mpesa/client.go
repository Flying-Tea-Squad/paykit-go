package mpesa

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
	"encoding/base64"
)

type Client struct {
	baseURL string
	httpClient *http.Client
	passkey string
}

func (c *Client) STKPush(ctx context.Context, req STKPushRequest) (*STKPushResponse, error){

	req.Timestamp = generateTimeStamp()
	req.Password = generatePassword(
		req.BusinessShortCode,
		c.passkey,
		req.Timestamp,
	)


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

	httpReq.Header.Set("Idempotency-Key", req.IdempotencyKey)


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

func generateTimeStamp() string{
	return time.Now().Format("20060102150405")
}

func generatePassword(shortcode, passkey, timeStamp string) string{
	raw := shortcode + passkey + timeStamp

	return base64.StdEncoding.EncodeToString([]byte(raw))
}