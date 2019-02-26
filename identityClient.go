package main

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"time"
)

type identityClient struct {
	*jsonHTTPClient
}

func newIdentityClient(rawBaseURL string) *identityClient {
	httpClient := &http.Client{
		Timeout: time.Millisecond * 500, //TODO: configure this...
	}

	baseURL, _ := url.Parse(rawBaseURL)
	jsonClient := &jsonHTTPClient{
		httpClient: httpClient,
		UserAgent:  "hacknights/tokens",
		BaseURL:    baseURL,
	}

	return &identityClient{
		jsonHTTPClient: jsonClient,
	}
}

func (c *identityClient) byCredentials(appID, user, pass string) (map[string]interface{}, error) {
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
	return c.authenticate(appID, auth)
}

func (c *identityClient) authenticate(appID, authorization string) (map[string]interface{}, error) {
	req, err := c.newRequest("GET", appID+"/authenticate", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", authorization)

	type result struct {
		OK      bool                   `json:"ok"`
		Errors  []string               `json:"errors,omitempty"`
		Content map[string]interface{} `json:"content,omitempty"`
	}
	res := result{}
	_, err = c.do(req, &res)
	return res.Content, err
}

func (c *identityClient) revision(appID, userID string) (string, error) {
	req, err := c.newRequest("GET", appID+"/users/"+userID+"/claims/revision", nil)
	if err != nil {
		return "", err
	}

	type result struct {
		OK      bool     `json:"ok"`
		Errors  []string `json:"errors,omitempty"`
		Content string   `json:"content,omitempty"`
	}
	res := result{}
	_, err = c.do(req, &res)
	return res.Content, err
}
