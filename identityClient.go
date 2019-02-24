package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type jsonHTTPClient struct {
	BaseURL   *url.URL
	UserAgent string

	httpClient *http.Client
}

func (c *jsonHTTPClient) newRequest(method, path string, body interface{}) (*http.Request, error) {
	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)
	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

func (c *jsonHTTPClient) do(req *http.Request, v interface{}) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || 300 <= resp.StatusCode {
		return nil, fmt.Errorf("invalid status: %s", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(v)
	return resp, err
}

type identityClient struct {
	*jsonHTTPClient
}

func newIdentityClient(rawBaseURL string) *identityClient {
	baseURL, _ := url.Parse(rawBaseURL)
	jsonClient := &jsonHTTPClient{
		httpClient: &http.Client{
			Timeout: time.Millisecond * 500, //TODO: configure this...
		},
		UserAgent: "hacknights/tokens",
		BaseURL:   baseURL,
	}
	return &identityClient{
		jsonHTTPClient: jsonClient,
	}
}

func (ic *identityClient) byCredentials(user, pass string) (map[string]interface{}, error) {
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
	return ic.authenticate(auth)
}

func (ic *identityClient) authenticate(authorization string) (map[string]interface{}, error) {
	req, err := ic.newRequest("GET", "api/authenticate", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", authorization)

	type result struct {
		Ok      bool                   `json:"ok"`
		Errors  []string               `json:"errors,omitempty"`
		Content map[string]interface{} `json:"content,omitempty"`
	}
	res := result{}
	_, err = ic.do(req, &res)
	return res.Content, err
}
