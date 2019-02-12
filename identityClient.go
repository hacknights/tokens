package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type identityClient struct {
}

func (ic *identityClient) ByCredentials(user, pass string) (*identity, error) {

	client := &http.Client{
		Timeout: time.Millisecond * 50, //TODO: configure this...
	}

	req, err := http.NewRequest("GET", "http://:8080/tokens", nil) //TODO: config
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(user, pass)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status: %s", resp.Status)
	}

	dec := json.NewDecoder(resp.Body)
	result := identity{}
	if err := dec.Decode(result); err != nil {
		return nil, err
	}
	return &result, nil
	//return &identity{}, nil
}

type identity struct {
}

func (i *identity) ID() string {
	return "anonymous"
}

func (i *identity) Claims() map[string]interface{} {
	return map[string]interface{}{
		"fn":   "Joe",
		"role": "basic",
	}
}
