package main

import (
	_ "expvar" //standardized metrics (GET /debug/vars)
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	handleInterrupts(make(chan os.Signal, 1))

	//TODO: read config

	app := newApp()
	always := use(traceIDs, performanceLogging, recuperate)

	s := &http.Server{ //TODO: TLS
		Addr:           ":8080", //TODO: use config
		Handler:        always(app.ServeHTTP),
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
