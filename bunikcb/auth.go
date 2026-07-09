package bunikcb

import (
	"net/http"
)

func (c *Client) GetAccessToken() (string, error) {
	// CREATED A REQUEST FROM BUNI SERVER
	// build full token url
	url := c.config.BaseURL + "/token?grant_type=client_credentials"

	// create http request
	req, err := http.NewRequest("POST", url, nil)

	if err != nil {
		return "", err
	}

	//verify the request
	// we call req.SetBasicAuth()before sending the request because BUni SErver needs to verify who is making the request
	req.SetBasicAuth(
		c.config.ConsumerKey
		c.config.ConsumerSecret
	)

	// send the request
	// Do() accepts a fully prepared http.Request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	// defer is used to close the body(close the connection) and prevent it from giving a continous response from BUNI
	// defered call will run before the fucntion exists
	defer resp.Body.Close()
}