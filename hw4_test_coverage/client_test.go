package main

import (
	"net/http"          // request, response, status codes
	"net/http/httptest" // test server
	"strings"           // prefix (starts with)
	"testing"           // test subsystem
	"time"              // duration, sleep (timeout)
)

// код писать тут
// ---

func TestLimitLessThanZero(t *testing.T) {
	// to check if the http request is going to be performed or not
	var serverCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		// response does not matter
	}))

	req := SearchRequest{Limit: -1} // limit is negative
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	}

	// error is present -> check the actual message
	if err.Error() != "limit must be > 0" {
		t.Errorf("Expected limit less than zero error, but received something else")
	}

	if serverCalled {
		t.Errorf("Expected the http request not to be made")
	}
}

func TestOffsetLessThanZero(t *testing.T) {
	// to check if the http request is going to be performed or not
	var serverCalled bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true
		// response does not matter
	}))

	req := SearchRequest{Offset: -1} // offset is negative
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertions
	if err == nil {
		t.Errorf("Error expected, none received")
	}

	// error is present -> check the actual message
	if err.Error() != "offset must be > 0" {
		t.Errorf("Expected offset less than zero error, but received something else")
	}

	if serverCalled {
		t.Errorf("Expected the http request not to be made")
	}
}

// ---
// http calls
func TestLimitRequestUpperValue(t *testing.T) {
	var isLimitCorrect = false // is the limit correct?

	// to check was server has received
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isLimitCorrect = (r.URL.Query().Get("limit") == "26") // and not 25 because of '++'
		// response does not matter
	}))

	req := SearchRequest{Limit: 30} // limit > 25 (25 is max)
	client := &SearchClient{URL: ts.URL}

	client.FindUsers(req) // main action (subject)

	// assertion(s)
	if !isLimitCorrect {
		t.Errorf("Expected the limit request value to be upper bounded by 25, but it was not")
	}
}

// ---

func TestTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // timeout is 1 second; -> sleep for 2 seconds
		// the actual response body/status does not matter that much
	}))

	defer ts.Close()

	req := SearchRequest{} // does not matter in this case
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if !strings.HasPrefix(err.Error(), "timeout for") {
			t.Errorf("Expected request timeout error, but received something else")
		}
	}
}

func TestClientDoUnknownError(t *testing.T) {
	req := SearchRequest{}    // does not matter in this case
	client := &SearchClient{} // notice no valid URL

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if !strings.HasPrefix(err.Error(), "unknown error") {
			t.Errorf("Expected Unknown Error, but received something else")
		}
	}
}

// 401, unauthenticated -> no token supplied
func TestUnauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized) // http status code 401
	}))

	defer ts.Close()

	req := SearchRequest{}               // does not matter in this case
	client := &SearchClient{URL: ts.URL} // <- no token

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if err.Error() != "Bad AccessToken" {
			t.Errorf("Expected Bad Access Token Error, but received something else")
		}
	}
}

// http 400, typically bad/missing request params
func TestScrambledErrorJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // http status code 400
		w.Write([]byte("notajson"))          // body (faulty)
	}))

	defer ts.Close()

	req := SearchRequest{} // does not matter in this case
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if !strings.HasPrefix(err.Error(), "cant unpack error json") { // starts with
			t.Errorf("Expected error json unpack error, but received something else")
		}
	}
}

// http 400, typically bad/missing request params
func TestBadOrderField(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)              // http status code 400
		w.Write([]byte(`{"error":"ErrorBadOrderField"}`)) // body
	}))

	defer ts.Close()

	req := SearchRequest{OrderField: "OK"} // field is not supported as sortable
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if err.Error() != "OrderFeld OK invalid" {
			t.Errorf("Expected order field error, but received something else")
		}
	}
}

// http 400, typically bad/missing request params
func TestUnknownError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)   // http status code 400
		w.Write([]byte(`{"error":"unknown"}`)) // body
	}))

	defer ts.Close()

	req := SearchRequest{OrderField: "-1"} // ?
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if err.Error() != "unknown bad request error: unknown" {
			t.Errorf("Expected unknown error, but received something else")
		}
	}
}

func TestScrambledJson(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http status code 200 (implicitly)
		w.Write([]byte("notajson")) // body
	}))

	defer ts.Close()

	req := SearchRequest{} // does not matter
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if !strings.HasPrefix(err.Error(), "cant unpack result json") {
			t.Errorf("Expected result json unpack error, but received something else")
		}
	}
}

// eq - equals
func TestNumberOfUsersEqLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http status code 200 (implicitly)
		users := `[
      {"id":1, "name": "john doe", "age": 30, "about": "love beer", "gender": "male"}
    ]`
		w.Write([]byte(users)) // body
	}))

	defer ts.Close()

	req := SearchRequest{Limit: 1} // request only one user
	client := &SearchClient{URL: ts.URL}

	res, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err != nil {
		t.Errorf("Error received while not expected")
	}

	// TODO: compare users
	if res.NextPage {
		t.Errorf("Next page should be false")
	}

	if len(res.Users) != 1 {
		t.Errorf("Exactly one user was expected")
	}
}

func TestNoUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http status code 200 (implicitly)
		w.Write([]byte(`[]`)) // body; empty array of users
	}))

	defer ts.Close()

	req := SearchRequest{Limit: 1} // request only one user
	client := &SearchClient{URL: ts.URL}

	res, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err != nil {
		t.Errorf("Error received while not expected")
	}

	// TODO: compare users
	if res.NextPage {
		t.Errorf("Next page should be false")
	}

	if len(res.Users) != 0 {
		t.Errorf("Zero users were expected")
	}
}

// gt - greater than
func TestNumberOfUsersGtLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http status code 200 (implicitly)
		users := `[
      {"id":1, "name": "john doe", "age": 30, "about": "love beer", "gender": "male"},
      {"id":2, "name": "homer simpson", "age": 34, "about": "hello", "gender": "male"}
    ]`
		w.Write([]byte(users)) // body
	}))

	defer ts.Close()

	req := SearchRequest{Limit: 1} // request only one user
	client := &SearchClient{URL: ts.URL}

	res, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err != nil {
		t.Errorf("Error received while not expected")
	}

	// TODO: compare users
	if !res.NextPage {
		t.Errorf("Next page should be true")
	}

	if len(res.Users) != 1 { // and not 2
		t.Errorf("Only one (=limit) user was expected")
	}
}

// ---

// 500, in general should not happen at all
func TestFatal(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError) // http status code 500
		// response body does not matter
	}))

	defer ts.Close()

	req := SearchRequest{} // does not matter in this case
	client := &SearchClient{URL: ts.URL}

	_, err := client.FindUsers(req) // main action (subject)

	// assertion(s)
	if err == nil {
		t.Errorf("Error expected, none received")
	} else {
		// error is present -> check the actual message
		if err.Error() != "SearchServer fatal error" {
			t.Errorf("Expected fatal error, but received something else")
		}
	}
}
