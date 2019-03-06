package main

import (
	// "bytes"
	"net/http"
)

// ----

// 1) receive incoming parameters
// 2) parse into appropriate structure
// 3) validate according to the structure -> reject if invalid
// 4) 

func userCreateHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()	// the first mux
	mux.HandleFunc("/create/", func (w http.ResponseWriter, r *http.Request){
		w.Write([]byte("Hello World"))
	})
}

// ---

