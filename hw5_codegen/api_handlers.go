package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// ----

// - receive incoming parameters
// - check the request method (GET/POST)
// - parse into appropriate structure (json -> structure)
// - validate according to the structure -> reject if invalid

// ---

type apiResponse struct {
	Error string 		 `json:"error"`
	Response interface{} `json:"response,omitempty"`	// any inner structure
}

// ---

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprint(w, "Request: " + r.URL.Path)

	ctx := r.Context()	// request context
	query := r.URL.Query()
	// TODO: mapping?
	age, _ := strconv.Atoi(query.Get("age"))
	params := CreateParams { query.Get("login"), query.Get("account_name"), query.Get("status"), age }
	// TODO: Structure validation
	err := params.Validate() // return first error or nil (no errors = valid)
	if err != nil {
		ar := apiResponse { err.Error(), nil }
		j, err := json.Marshal(ar)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return	
		}
		w.Write(j)
		return
	}

	// ---

	srvResponse, err := srv.Create(ctx, params)	// the main call

	var ar apiResponse
	if err != nil {
		// error
		e := err.(ApiError)	// type assertion; convert to the ~correct type (ApiError struct is allowed to be used)
		w.WriteHeader(e.HTTPStatus)	// -> correct HTTP error status code
		ar = apiResponse { e.Err.Error(), nil }	// new api response
	} else {
		// success
		ar = apiResponse { "", srvResponse } // blank error to be present inside
	}

	// ---
	j, err := json.Marshal(ar)	// -> json
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return		
	}
	w.Write(j)
}

// ---

func (cp CreateParams) Validate() error {
	// presence
	if len(cp.Login) <= 0 {
		return fmt.Errorf("login must me not empty")	// 'be' -> 'me' typo
	}
	
	// min len (str)
	if len(cp.Login) < 10 {
		return fmt.Errorf("login len must be >= 10")
	}

	// TODO: continue
	// ... (to be continued)

	return nil	// all valid
}