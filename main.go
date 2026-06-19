package main

import "net/http"

func main() {
	servemux := http.NewServeMux()
	servemux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	servemux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	server := http.Server{Handler: servemux, Addr: ":8080"}
	server.ListenAndServe()
}
