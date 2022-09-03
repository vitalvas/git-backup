package api

import (
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"time"
)

func RunAPIServer() {
	r := http.NewServeMux()

	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := http.Server{
		Addr:              os.Getenv("API_SERVER_ADDR"),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
