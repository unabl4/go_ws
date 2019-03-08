package main

import (
	"bytes"
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

// json serializer
func encodeJson(content interface{}) ([]byte, error) {
	b := &bytes.Buffer{}
	c := json.NewEncoder(b)	// new json encoder
	c.SetEscapeHTML(false)
	err := c.Encode(content)	// -> json

	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func throwBadRequest(w http.ResponseWriter, errorMessage string) {
	w.WriteHeader(http.StatusBadRequest)	// 400 (bad request)

	ar := apiResponse { errorMessage, nil }
	j, err := encodeJson(ar)
	
	if err != nil {	// json encoding error
		http.Error(w, err.Error(), http.StatusInternalServerError)	// json serialization error
		return		
	}

	w.Write(j)	// flush
}

// ---

// the main router (for MyApi)
func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
    case "/user/create":
        srv.handlerUserCreate(w,r)
    default:
		// 404
		http.NotFound(w,r)
    }
}

// ====
// handler functions (inner)

func (srv *MyApi) handlerUserCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()	// request context
	query := r.URL.Query()	// primary data source
	// STRUCT
	// query -> struct extraction
	params := CreateParams {}
	params.Login  = query.Get("login")	// LOWERCASE!
	params.Name   = query.Get("account_name")
	params.Status = query.Get("status")

	// INTEGER extraction special case
	rawAge := query.Get("age")
	if len(rawAge) > 0 {	// isset?
		ageInt, err := strconv.Atoi(query.Get("age"))	// -> int conversion example

		// special case for integers
		if err != nil {
			throwBadRequest(w, "age must be integer")
			return	// stop
		}

		params.Age = ageInt	// the final attribution
	}

	// TODO: Structure validation
	err := params.Validate() // return first error or nil (no errors = valid)
	if err != nil {
		// input params are NOT valid -> BAD REQUEST
		throwBadRequest(w, err.Error())
		return	// stop
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

	j,err := encodeJson(ar)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)	// json serialization error
		return		
	}

	w.Write(j)
}

// ---

func (cp CreateParams) Validate() error {
	// presence
	if len(cp.Login) <= 0 {
		return fmt.Errorf("login must me not empty")	// 'be' -> 'me' typo; but we keep it to pass the tests
	}
	
	// min len (str)
	if len(cp.Login) < 10 {
		return fmt.Errorf("login len must be >= 10")
	}

	// TODO: continue
	// ... (to be continued)

	return nil	// all valid
}