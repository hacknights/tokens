package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// at a glance routes directs us where to look. defines middleware on routes
func (a *app) routes() {

	always := use(performanceLogging, recuperate)

	a.HandleFunc(http.MethodGet, "/tokens", a.handleTokens(signKey)).With(basicAuth(a.identity.ByCredentials), always)
	a.HandleFunc(http.MethodGet, "/refresh", a.handleRefresh()).With(jwtAuth(), always)
	a.HandleFunc(http.MethodGet, "/revoke", a.handleRevoke()).With(jwtAuth(), always)
	a.HandleFunc(http.MethodGet, "/revoke-all", a.handleRevokeAll()).With(jwtAuth(), always)

	a.HandleFunc(http.MethodGet, "/restricted", a.handleRevokeAll()).With(jwtAuth(), always)
}

func use(middleware ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(h http.HandlerFunc) http.HandlerFunc {
		for _, m := range middleware {
			h = m(h)
		}
		return h
	}
}

type authenticatorFunc func(user, pass string) (*identity, error)

// basicAuth validates Basic credentials and populates identity in context
//
// Example:
//   http.Handle("/", basicAuth(a.identity.ByCredentials)(r))
func basicAuth(authenticate authenticatorFunc) func(h http.HandlerFunc) http.HandlerFunc {

	setIdentity := func(ctx context.Context, id *identity) {

	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("WWW-Authenticate", `Basic realm="Fail"`)

			s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			if len(s) != 2 {
				http.Error(w, "Not authorized", http.StatusUnauthorized)
				return
			}

			b, err := base64.StdEncoding.DecodeString(s[1])
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			pair := strings.SplitN(string(b), ":", 2)
			if len(pair) != 2 {
				http.Error(w, "Not authorized", http.StatusUnauthorized)
				return
			}

			fmt.Println(pair[0], pair[1])
			identity, err := authenticate(pair[0], pair[1])
			if err != nil {
				http.Error(w, "Not authorized", http.StatusUnauthorized)
				return
			}

			setIdentity(r.Context(), identity)

			next.ServeHTTP(w, r)
		}
	}
}

// jwtAuth validates a jwt and populates identity in context
//
// Example:
//   http.Handle("/", jwtAuth(jwtValueFunc, jwt.Keyfunc)(r))
func jwtAuth() func(h http.HandlerFunc) http.HandlerFunc {

	setTokenIdentity := func(ctx context.Context, token *jwt.Token) {
		// set token
		// set Identity from Claims
	}

	jwtValue := func(r *http.Request) (string, error) {
		s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
		if len(s) != 2 {
			return "", fmt.Errorf("missing authorization header")
		}
		return s[1], nil
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			//Only accept expected valid signing methods - specifically, don't accept None
			log.Printf("Unexpected signing method: %v\n", token.Header["alg"])
			return nil, fmt.Errorf("invalid method")
		}
		//TODO: use token.Header["kid"]
		return verifyKey, nil
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			value, err := jwtValue(r)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				log.Printf("jwtValue: %+v\n", err)
				return
			}

			token, err := jwt.Parse(value, keyFunc)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				log.Printf("jwtParse: %+v\n", err)
				return
			}

			//TODO: verify required claims

			setTokenIdentity(r.Context(), token)

			next.ServeHTTP(w, r)
		}
	}
}

// performanceLogging writes the RequestURI and duration of handlers
//
// Example:
//   http.Handle("/", performanceLogging(r))
func performanceLogging(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			log.Printf("%s\t%s", elapsed.String(), r.RequestURI)
		}()
		next.ServeHTTP(w, r)
	})
}

// recuperate catches panics in handlers, logs the stack trace and serves an HTTP 500 error.
//
// Example:
//   http.Handle("/", recuperate(r))
func recuperate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "HTTP 500: internal server error (we've logged it!)", http.StatusInternalServerError)
				log.Printf("Handler panicked: %s\n%s", err, debug.Stack())
			}
		}()
		next.ServeHTTP(w, r)
	}
}
