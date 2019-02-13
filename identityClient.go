package main

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type identityClient struct {
}

func (ic *identityClient) ByCredentials(user, pass string) (*identity, error) {
	type result struct {
		Ok      bool     `json:"ok"`
		Errors  []string `json:"errors,omitempty"`
		Content identity `json:"content,omitempty"`
	}

	client := &http.Client{
		Timeout: time.Millisecond * 50, //TODO: configure this...
	}

	req, err := http.NewRequest("GET", "http://:8080/authenticate", nil) //TODO: config
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
	res := result{}
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}
	return &res.Content, nil
}

type identity struct {
	ID     string                 `json:"id,omitempty"`
	Claims map[string]interface{} `json:"claims,omitempty"`
}

//TODO: Move this method onto identity
func (u *identity) GenerateTokens(signKey *rsa.PrivateKey) (interface{}, int, error) {

	access, refresh, err := createTokens(u, signKey) //todo: attach this to handler (don't need signKey)
	if err != nil {
		fmt.Printf("Token Signing error: %v\n", err)                                  //TODO: use handler's logger
		return nil, http.StatusInternalServerError, fmt.Errorf("error signing token") //TODO: compose errors
	}

	//TODO: save access claims by refresh.jti
	//TODO: refresh.jti by refresh.sub

	return struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}{
		access,
		refresh,
	}, http.StatusOK, nil
}
