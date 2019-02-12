package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type negotiatorFunc func(w http.ResponseWriter, r *http.Request) func(interface{}, int, error)

func negotiator(w http.ResponseWriter, r *http.Request) func(interface{}, int, error) {
	return func(value interface{}, status int, err error) {

		if err != nil {
			w.WriteHeader(status)
			fmt.Fprint(w, err)
			return
		}

		if value == nil {
			w.WriteHeader(status)
			return
		}

		json, e := json.Marshal(value)
		if e != nil {
			http.Error(w, "unable to negotiate response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json") //determine from Accept
		w.WriteHeader(status)
		w.Write(json)
	}
}
