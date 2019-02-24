package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type negotiatorFactory func(w http.ResponseWriter, r *http.Request) negotiatorFunc

type negotiatorFunc func(interface{}, int, error)

func (n negotiatorFunc) ok(value interface{}) {
	n(value, http.StatusOK, nil)
}

func (n negotiatorFunc) notFound() {
	n(nil, http.StatusNotFound, fmt.Errorf("not found"))
}

func (n negotiatorFunc) internalServer(err string) {
	n.internalServerError(fmt.Errorf(err))
}

func (n negotiatorFunc) internalServerError(err error) {
	n(nil, http.StatusInternalServerError, err)
}

func (n negotiatorFunc) unauthorized(err string) {
	n.unauthorizedError(fmt.Errorf(err))
}
func (n negotiatorFunc) unauthorizedError(err error) {
	n(nil, http.StatusUnauthorized, err)
}

func negotiator(w http.ResponseWriter, r *http.Request) negotiatorFunc {
	type result struct {
		Ok      bool        `json:"ok"`
		Errors  []string    `json:"errors,omitempty"`
		Content interface{} `json:"content,omitempty"`
	}
	newResult := func(value interface{}, status int, err error) result {
		res := result{
			Ok:      err == nil,
			Content: value,
		}
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
		}
		return res
	}

	return func(value interface{}, status int, err error) {

		res := newResult(value, status, err)

		json, e := json.Marshal(res)
		if e != nil {
			http.Error(w, "unable to negotiate response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(json)
	}
}
