package main

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
)

type tokensHandler struct {
	negotiator negotiatorFunc
	signKey    *rsa.PrivateKey
	identity   *identityClient
}

func (h *tokensHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var p string
	p, r.URL.Path = shiftPath(r.URL.Path)
	n := h.negotiator(w, r)

	if p != "" {
		n(nil, http.StatusNotFound, fmt.Errorf("not found"))
		return
	}

	switch r.Method {
	case "GET":

		w.Header().Set("WWW-Authenticate", `Basic realm="tokens"`) //TODO: required?

		user, pass, ok := r.BasicAuth()
		if !ok {
			n(nil, http.StatusUnauthorized, fmt.Errorf("missing auth"))
			return
		}

		u, err := h.identity.ByCredentials(user, pass)
		if err != nil {
			n(nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials"))
			return
		}

		n(h.handleGet(u))

	default:
		n(nil, http.StatusMethodNotAllowed, fmt.Errorf("Only GET is allowed"))
		return
	}
}

//TODO: While ServeHttp seems noisy, I like that this method is mostly http independent (could be domain?)
func (h *tokensHandler) handleGet(u *identity) (interface{}, int, error) {

	access, refresh, err := createTokens(u, h.signKey) //todo: attach this to handler (don't need signKey)
	if err != nil {
		log.Printf("Token Signing error: %v\n", err)                                  //TODO: use handler's logger
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
