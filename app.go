package main

import (
	"context"
	"net/http"

	"github.com/nescio/route66"
)

type app struct {
	router   *route66.Router
	db       string //buntdb (simple, has TTLs)
	identity *identityClient
}

func newApp() *app {
	r := route66.NewRouter()
	//TODO: r.NotFound how to wrap this in middleware (performanceLogging, recuperate)?
	a := &app{
		router: r,
	}
	a.routes()
	return a
}

func (a *app) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

type route interface {
	With(middleware ...func(http.HandlerFunc) http.HandlerFunc)
}

func (a *app) HandleFunc(method, pattern string, fn http.HandlerFunc) route {
	return a.router.HandleFunc(method, pattern, fn)
}

func (a *app) routeParam(ctx context.Context, param string) string {
	return route66.Param(ctx, param)
}
