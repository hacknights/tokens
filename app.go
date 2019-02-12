package main

import (
	"fmt"
	"net/http"
)

type app struct {
	negotiator    negotiatorFunc
	tokensHandler *tokensHandler
	db            string //buntdb (simple, has TTLs)
}

func newApp() *app {
	return &app{
		negotiator: negotiator,
		tokensHandler: &tokensHandler{
			negotiator: negotiator,
			identity:   &identityClient{},
			signKey:    signKey,
		},
	}
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var p string
	p, r.URL.Path = shiftPath(r.URL.Path)

	if p == "tokens" {
		a.tokensHandler.ServeHTTP(w, r)
		return
	}
	a.negotiator(w, r)(nil, http.StatusNotFound, fmt.Errorf("not found"))
}
