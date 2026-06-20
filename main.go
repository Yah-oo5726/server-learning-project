package main

import (
	"net/http"
	"strconv"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int64
}

func (c *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func main() {
	config := apiConfig{}
	servemux := http.NewServeMux()
	apphandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/app", http.FileServer(http.Dir("."))).ServeHTTP(w, r)
	})
	servemux.Handle("/app/", config.middlewareMetricsInc(apphandler))
	servemux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	servemux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hits: " + strconv.FormatInt(config.fileserverHits.Load(), 10)))
	})
	servemux.HandleFunc("POST /reset", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		config.fileserverHits.Store(0)
		w.Write([]byte("Hits reset to 0"))
	})
	server := http.Server{Handler: servemux, Addr: ":8080"}
	server.ListenAndServe()
}
