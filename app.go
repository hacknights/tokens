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
	authenticate func(appID, authorization string) (map[string]interface{}, error)
	getRevision  func(appID, userID string) (string, error)
	Issuer       string
}

type app struct {
	negotiator negotiator.Factory

	authJWT      func(h http.HandlerFunc) http.HandlerFunc
	authByScheme func(h http.HandlerFunc) http.HandlerFunc

	generateTokens tokenGeneratorFunc
	authenticate   func(appID, authorization string) (map[string]interface{}, error)
	getRevision    func(appID, userID string) (string, error)
}

func newAppHandler(cfg appConfig) http.Handler {
	tokenGenerator := mustNewTokenGenerator(cfg.PrivKeyBytes)

	authJWT := middleware.AuthJWT(
		middleware.FromHeader,
		func(kid string) []byte { return cfg.PubKeyBytes },
		cfg.Issuer, //required audience
	)

	authByScheme := middleware.AuthByScheme(map[string]func(h http.HandlerFunc) http.HandlerFunc{
		"basic": middleware.AuthSkip(),
	})

	a := app{
		negotiator: negotiator.NewNegotiator,

		authJWT:      authJWT,
		authByScheme: authByScheme,

		generateTokens: tokenGenerator,
		authenticate:   cfg.authenticate,
		getRevision:    cfg.getRevision,
	}

	return middleware.Use(
		middleware.TraceIDs,
		middleware.Logging,
		middleware.Recuperate)(a.ServeHTTP)
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const (
		tokens       string = "/tokens"
		refresh      string = "/refresh"
		jwks         string = "/.well-known/openid-configuration/jwks" //TODO: provide an endpoint for Validator to lookup the PublicKey (must be over https)
		authenticate string = "/:appid/authenticate"                   //TODO: usery must capture appId
	)
	match := firstCaseInsensitivePrefixMatch(r.URL.Path, tokens, refresh, authenticate)

	switch match {
	case tokens:
		r.URL.Path = strings.TrimPrefix(r.URL.Path, tokens)
		a.authByScheme(a.handleTokens)(w, r)

	case refresh:
		r.URL.Path = strings.TrimPrefix(r.URL.Path, refresh)
		a.authJWT(a.handleRefresh)(w, r)

	case authenticate:
		//TODO: should be handled by usery (for now return ok)
		//TODO: Endure usery validates AuthBasic
		//TODO: How does usery validate RefreshTokens (the audience won't match)
		a.authByScheme(a.handleAuthenticate)(w, r)

	default:
		http.DefaultServeMux.ServeHTTP(w, r) //TOOD: should this be passed in
	}
}
func (a *app) handleTokens(w http.ResponseWriter, r *http.Request) {
	appID, _ := shiftPath(r.URL.Path)
	n := a.negotiator(w, r)

	if http.MethodGet != r.Method || appID == "" {
		n.NotFound()
		return
	}

	auth := r.Header.Get("Authorization")

	u, err := a.authenticate(appID, auth)
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

func (a *app) handleRefresh(w http.ResponseWriter, r *http.Request) {
	head, _ := shiftPath(r.URL.Path)
	n := a.negotiator(w, r)

	if http.MethodGet != r.Method || head != "" {
		n.NotFound()
		return
	}

	//TODO: everything we need should be in the RefreshToken
	//Must verify rev matches RefreshToken.Rev
	rev, err := a.getRevision("appID", "userID")
	if err != nil || rev != "RefreshToken.Rev" {
		n.Unauthorized("invalid credentials")
		return
	}

	var claims map[string]interface{} //from RefreshToken
	tokens, err := a.generateTokens(claims)
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
			"rev": "0",
		},
	)
}
