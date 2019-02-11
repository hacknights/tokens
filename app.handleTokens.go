package main

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
)

//define request/response inside the handler

func (a *app) handleTokens(signKey *rsa.PrivateKey) http.HandlerFunc {
	type req struct {
		Name string
	}
	type resp struct {
		Access  string `json:"access"`
		Refresh string `json:"refresh"`
	}
	return func(w http.ResponseWriter, r *http.Request) {

		u := &identity{} //parse from context

		access, refresh, err := createTokens(u, signKey)
		if err != nil {
			http.Error(w, "error signing token", http.StatusInternalServerError)
			log.Printf("Token Signing error: %v\n", err)
			return
		}
		resp := resp{
			Access:  access,
			Refresh: refresh,
		}

		//TODO: save access claims by refresh.jti
		//TODO: refresh.jti by refresh.sub

		//TODO: a.Negotiate(w, r, http.StatusOK, resp)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"access":"%s","refresh":"%s"}`, resp.Access, resp.Refresh)
	}
}
