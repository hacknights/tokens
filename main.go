package main

import (
	_ "expvar" //standardized metrics (GET /debug/vars)
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	handleInterrupts(make(chan os.Signal, 1))

	privKeyBytes, err := ioutil.ReadFile("keys/app.rsa")
	fatal(err)

	pubKeyBytes, err := ioutil.ReadFile("keys/app.rsa.pub")
	fatal(err)

	ic := newIdentityClient("http://:8081/") //TODO: This should get /user/:uid/claims and needs its own credentials...

	//TODO: with config
	app := newAppHandler(
		appConfig{
			PrivKeyBytes: privKeyBytes,
			PubKeyBytes:  pubKeyBytes,
			Issuer:       "https://auth.hacknights.club",

			authenticate: ic.authenticate, //TODO: circuit-break to anonymous?
			getRevision:  ic.revision,
		},
	)

	s := &http.Server{ //TODO: TLS
		Addr:           ":8081", //TODO: use config
		Handler:        app,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, //1MB
	}

	log.Printf("Listening... %s\n", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleInterrupts(ch chan os.Signal) {
	signal.Notify(ch, os.Interrupt)
	go func() {
		for sig := range ch {
			fmt.Printf("Exit... %v\n", sig)
			ch = nil
			os.Exit(1)
		}
	}()
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
