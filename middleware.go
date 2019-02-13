package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

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

			//TODO: is this required?
			w.Header().Set("WWW-Authenticate", `Basic realm="Fail"`)

			user, pass, ok := r.BasicAuth()
			if !ok {
				http.Error(w, "Not authorized", http.StatusUnauthorized)
				return
			}

			identity, err := authenticate(user, pass)
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

			tokenString, err := jwtValue(r)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				log.Printf("jwtValue: %+v\n", err)
				return
			}

			jwt.ParseWithClaims(
				tokenString,
				jwt.StandardClaims{
					Issuer:   "",
					Audience: "",
				},
				keyFunc,
			)
			//ParseWithClaims
			token, err := jwt.Parse(tokenString, keyFunc)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				log.Printf("jwtParse: %+v\n", err)
				return
			}

			token.Claims.Valid()

			//TODO: verify required claims

			setTokenIdentity(r.Context(), token)

			next.ServeHTTP(w, r)
		}
	}
}

// traceIDs writes the RequestID, and possibly CorrelationID, in the Request header
//
// Example:
//   http.Handle("/", traceIDs(r))
func traceIDs(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r2 := new(http.Request)
		*r2 = *r
		rid := uuid.New().String()
		r2.Header.Set("X-Request-Id", rid)
		//TODO: if not already set, copy RequestID to CorrelationID...?
		next(w, r2)
	})
}

// performanceLogging writes the RequestURI and duration of handlers
//
// Example:
//   http.Handle("/", performanceLogging(r))
func performanceLogging(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := newLoggingResponseWriter(w)
		start := time.Now()
		defer func() {
			elapsed := time.Since(start)
			rid := r.Header["X-Request-Id"]
			log.Printf("%s\t%3d\t%-7d\t%s\t%s", rid, l.statusCode, l.length, elapsed.String(), r.RequestURI)
		}()
		next.ServeHTTP(l, r)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	length     int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		length:         0}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	n, err := lrw.ResponseWriter.Write(b)
	lrw.length += n
	return n, err
}

// recuperate catches panics in handlers, logs the stack trace and serves an HTTP 500 error.
//
// Example:
//   http.Handle("/", recuperate(r))
func recuperate(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, "HTTP 500: internal server error (we've logged it!)", http.StatusInternalServerError)
				log.Printf("Handler panicked: %s\n%s", err, debug.Stack())
			}
		}()
		next.ServeHTTP(w, r)
	})
}
