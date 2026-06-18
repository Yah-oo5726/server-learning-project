package main

import "net/http"

func main() {
	servemux := http.NewServeMux()
	server := http.Server{Handler: servemux, Addr: ":8080"}
	server.ListenAndServe()
}
