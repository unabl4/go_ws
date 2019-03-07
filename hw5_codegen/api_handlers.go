package main

import (
	"encoding/json"
	"net/http"
)

// ----

// - receive incoming parameters
// - check the request method (GET/POST)
// - parse into appropriate structure (json -> structure)
// - validate according to the structure -> reject if invalid

// ---

type ApiResponse struct {
	Error string 		 `json:"error"`
	Response interface{} `json:"response,omitempty"`	// any inner structure
}

// ---

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprint(w, "Request: " + r.URL.Path)

	ctx := r.Context()	// request context
	params := CreateParams { }
	// TODO: Structure validation
	// the main call
	newUser, err := srv.Create(ctx, params)

	if err != nil {
		// error
		e := err.(ApiError)	// type assertion; convert to the ~correct type (ApiError struct is allowed to be used)
		w.WriteHeader(e.HTTPStatus)	// -> correct HTTP error status code

		r := ApiResponse { e.Err.Error(), nil }	// new api response
		j, err := json.Marshal(r) // -> json
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return		
		}
		w.Write(j)
	} else {
		// success
		r := ApiResponse { "", newUser } // blank error to be present inside
		j, err := json.Marshal(r)	// -> json
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return		
		}
		w.Write(j)
	}
}