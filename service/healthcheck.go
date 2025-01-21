package service

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Healthchecker struct {
	Server http.Server
}

func NewHealthchecker(healthcheckPort int) Healthchecker {
	mux := http.NewServeMux()
	mux.Handle("/", handleHealthcheck())
	return Healthchecker{
		Server: http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", healthcheckPort),
			Handler: mux,
		},
	}
}

func handleHealthcheck() http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Debug("received healthcheck request")
			// This will have a status of 200
			fmt.Fprintf(w, "all good in the hood")
		},
	)
}
