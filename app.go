package main

import (
	"net/http"

	"github.com/hacknights/middleware"
)

type app struct {
	negotiator negotiatorFactory

	authenticate func(authorization string) (map[string]interface{}, error)
	createTokens func(claims map[string]interface{}) (*tokens, error)
	authJWT      func(h http.HandlerFunc) http.HandlerFunc
}

func newAppHandler() http.Handler {
	ic := newIdentityClient("http://:8080/")

	a := app{
		negotiator: negotiator,

		authenticate: ic.authenticate,
		authJWT:      middleware.AuthJWT(verifyKey),

		createTokens: func(claims map[string]interface{}) (*tokens, error) {
			return createTokens(claims, signKey)
		},
	}

	return middleware.Use(
		middleware.TraceIDs,
		middleware.Logging,
		middleware.Recuperate)(a.ServeHTTP)
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	head, rest := shiftPath(r.URL.Path)

	switch head {
	case "authenticate":
		//TODO: should be handled by usery (for now return ok)
		// Auth is either Basic or Jwt-Refresh
		a.negotiator(w, r).ok(
			map[string]string{
				"sub": "abc123",
			},
		)
	case "restricted":
		//TODO: Remove this - only used to validate the AuthJWT middleware
		a.authJWT(func(w http.ResponseWriter, r *http.Request) {})(w, r)
	case "tokens":
		r.URL.Path = rest
		a.handleGetTokens(w, r)
	default:
		http.DefaultServeMux.ServeHTTP(w, r) //TOOD: should this be passed in
	}
}

func (a *app) handleGetTokens(w http.ResponseWriter, r *http.Request) {
	n := a.negotiator(w, r)
	var head string
	head, r.URL.Path = shiftPath(r.URL.Path)

	if r.Method != http.MethodGet || head != "" {
		n.notFound()
		return
	}

	auth := r.Header.Get("Authorization")

	u, err := a.authenticate(auth)
	if err != nil {
		n.unauthorized("invalid credentials")
		return
	}

	tokens, err := a.createTokens(u)
	if err != nil {
		n.internalServer("error signing token")
		return
	}

	n.ok(tokens)
}
