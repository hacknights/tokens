package main

import (
	"encoding/json"
	"net/http"
)

type negotiatorFunc func(w http.ResponseWriter, r *http.Request) func(interface{}, int, error)

func negotiator(w http.ResponseWriter, r *http.Request) func(interface{}, int, error) {
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

		w.Header().Set("Content-Type", "application/json") //determine from Accept
		w.WriteHeader(status)
		w.Write(json)
	}
}
