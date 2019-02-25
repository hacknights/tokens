package main

import (
	"net/http"
	"strings"

	"github.com/hacknights/middleware"
	"github.com/hacknights/negotiator"
)

type appConfig struct {
	PrivKeyBytes []byte // openssl genrsa -out app.rsa keysize
	PubKeyBytes  []byte // openssl rsa -in app.rsa -pubout > app.rsa.pub
	authenticate func(authorization string) (map[string]interface{}, error)
	Issuer       string
}

type app struct {
	negotiator negotiator.Factory

	authJWT      func(h http.HandlerFunc) http.HandlerFunc
	authByScheme func(h http.HandlerFunc) http.HandlerFunc

	generateTokens tokenGeneratorFunc
	authenticate   func(authorization string) (map[string]interface{}, error)
}

func newAppHandler(cfg appConfig) http.Handler {
	tokenGenerator := mustNewTokenGenerator(cfg.PrivKeyBytes)

	authJWT := middleware.AuthJWT(
		middleware.FromHeader,
		func(kid string) []byte { return cfg.PubKeyBytes },
		cfg.Issuer, //required audience
	)

	authByScheme := middleware.AuthByScheme(map[string]func(h http.HandlerFunc) http.HandlerFunc{
		"basic":  middleware.AuthSkip(), //TODO: DANGER this should be specified closer to usage, to avoid misue
		"bearer": authJWT,
	})

	a := app{
		negotiator: negotiator.NewNegotiator,

		authJWT:      authJWT,
		authByScheme: authByScheme,

		generateTokens: tokenGenerator,
		authenticate:   cfg.authenticate,
	}

	return middleware.Use(
		middleware.TraceIDs,
		middleware.Logging,
		middleware.Recuperate)(a.ServeHTTP)
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const (
		tokens       string = "/tokens"
		authenticate string = "/api/authenticate"
		jwks         string = "/.well-known/openid-configuration/jwks" //TODO: provide an endpoint for Validator to lookup the PublicKey (must be over https)
	)
	match := firstCaseInsensitivePrefixMatch(r.URL.Path, tokens, authenticate)

	switch match {
	case tokens:
		r.URL.Path = strings.TrimPrefix(r.URL.Path, tokens)
		a.authByScheme(a.handleGetTokens)(w, r)

	case authenticate:
		//TODO: should be handled by usery (for now return ok)
		//TODO: Endure usery validates AuthBasic
		//TODO: How does usery validate RefreshTokens (the audience won't match)
		a.authByScheme(a.handleAuthenticate)(w, r)

	default:
		http.DefaultServeMux.ServeHTTP(w, r) //TOOD: should this be passed in
	}
}
func (a *app) handleGetTokens(w http.ResponseWriter, r *http.Request) {
	head, _ := shiftPath(r.URL.Path)
	n := a.negotiator(w, r)

	if http.MethodGet != r.Method || head != "" {
		n.NotFound()
		return
	}

	auth := r.Header.Get("Authorization")

	u, err := a.authenticate(auth)
	if err != nil {
		n.Unauthorized("invalid credentials")
		return
	}

	tokens, err := a.generateTokens(u)
	if err != nil {
		n.InternalServer("error signing token")
		return
	}

	n.OK(tokens)
}

func (a *app) handleAuthenticate(w http.ResponseWriter, r *http.Request) {
	//TODO: replace this with a circuit-breaker
	a.negotiator(w, r).OK(
		map[string]string{
			"sub": "anonymous",
		},
	)
}
