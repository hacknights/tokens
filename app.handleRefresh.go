package main

import (
	"net/http"
)

//define request/response inside the handler

func (a *app) handleRefresh() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}