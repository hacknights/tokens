package main

import (
	"crypto/rsa"
	"fmt"
	"net/http"
)

type app struct {
	negotiator negotiatorFunc
	db         string //buntdb (simple, has TTLs)

	identity *identityClient
	signKey  *rsa.PrivateKey
}

func newApp() *app {
	return &app{
		negotiator: negotiator,
		identity:   &identityClient{},
		signKey:    signKey,
	}
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p, path := shiftPath(r.URL.Path)

	switch p {
	case "tokens":
		r.URL.Path = path
		a.handleTokens(w, r)
		return
	case "authenticate":
		//TODO: should be handled by usery (for now return ok)
		a.negotiator(w, r)(
			identity{
				ID: "it works!",
			},
			http.StatusOK,
			nil,
		)
		return
	default:
		http.DefaultServeMux.ServeHTTP(w, r) //TOOD: should this be passed in
		return
	}
}

func (a *app) handleTokens(w http.ResponseWriter, r *http.Request) {
	var p string
	p, r.URL.Path = shiftPath(r.URL.Path)
	n := a.negotiator(w, r)

	if r.Method != "GET" {
		n(nil, http.StatusMethodNotAllowed, fmt.Errorf("Only GET is allowed"))
		return
	}

	switch p {
	case "":
		a.handleGetTokens(w, r)
	case "refresh":
		a.handleGetRefresh(w, r)
	case "revoke":
		a.handleGetRevoke(w, r)
	case "revoke-all":
		a.handleGetRevokeAll(w, r)
	default:
		n(nil, http.StatusNotFound, fmt.Errorf("not found"))
		return
	}
}

func (a *app) handleGetTokens(w http.ResponseWriter, r *http.Request) {
	n := a.negotiator(w, r)
	w.Header().Set("WWW-Authenticate", `Basic realm="tokens"`) //TODO: required?

	user, pass, ok := r.BasicAuth()
	if !ok {
		n(nil, http.StatusUnauthorized, fmt.Errorf("missing auth"))
		return
	}

	//TODO: suppress IdentityClient - create Tokens.ByCredentials(user, pass, signKey)
	u, err := a.identity.ByCredentials(user, pass)
	if err != nil {
		n(nil, http.StatusUnauthorized, fmt.Errorf("invalid credentials"))
		return
	}

	n(u.GenerateTokens(a.signKey))
}

func (a *app) handleGetRefresh(w http.ResponseWriter, r *http.Request) {}

func (a *app) handleGetRevoke(w http.ResponseWriter, r *http.Request) {}

func (a *app) handleGetRevokeAll(w http.ResponseWriter, r *http.Request) {}
