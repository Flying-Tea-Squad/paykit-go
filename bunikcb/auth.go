package bunikcb

import (
	"net/http"
	"fmt"
	"io"
)

func (c *Client) GetAccessToken() (string, error) {
	// CREATED A REQUEST FROM BUNI SERVER
	// build full token url
	url := c.config.BaseURL + "/token?grant_type=client_credentials"

	// create http request
	req, err := http.NewRequest(http.MethodPost, url, nil)

	if err != nil {
		return "", err
	}

	//verify the request
	// we call req.SetBasicAuth()before sending the request because BUni SErver needs to verify who is making the request
	req.SetBasicAuth(
		c.config.ConsumerKey,
		c.config.ConsumerSecret,
	)

	// send the request
	// Do() accepts a fully prepared http.Request
	resp, err := c.httpClient.Do(req)

	// occurs when the request is sent succesfully and has other issues
	if resp.StatusCode != http.StatusOK {
		//return "", errors.New("request failed")
		fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	// err only represents errors that occurred while Go was trying to send the HTTP request
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println(string(body))
	// defer is used to close the body(close the connection) and prevent it from giving a continous response from BUNI
	// defered call will run before the fucntion exists
	defer resp.Body.Close()
	
	return "", nil
}