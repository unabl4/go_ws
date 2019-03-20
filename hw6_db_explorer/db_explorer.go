package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

type Handler struct {
	DB *sql.DB
	// ---
	Tables []Table
}

type Table struct {
	Name string
}

// ===

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	h := Handler{} // new instance
	h.DB = db
	err := h.Initialize()
	if err != nil {
		return nil, err
	}
	return &h, nil // at the moment, not entirely sure what the error is for
}

// ===

func (h *Handler) Initialize() error {
	var n string                        // table name
	q, err := h.DB.Query("SHOW TABLES") // and not 'QueryRow'
	if err != nil {
		return err
	}

	defer h.DB.Close()
	for q.Next() { // loop through rows
		err := q.Scan(&n)
		if err != nil {
			return err
		}

		t := Table{n}
		h.Tables = append(h.Tables, t)
	}

	return nil // no errors
}

// ---

// api response structure
type Response struct {
	Error    string      `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
	// Records string
}

// ---

type DbRequest struct {
	TableName string // for what table
	// optional params
	RecordId *string // particular record we are after
	Limit    *string
	Offset   *string
}

// extract necessary params from the request url (table name, record id) path and query (limit, offset)
func parseDbRequest(r *http.Request) DbRequest {
	p := strings.Split(r.URL.Path, "/") // path components
	q := r.URL.Query()                  // query components
	d := DbRequest{}
	var t []string
	for _, h := range p { // filter out empty parts
		if h != "" {
			t = append(t, h)
		}
	}

	if len(t) > 0 {
		// any path params?
		d.TableName = t[0]
		if len(t) > 1 {
			d.RecordId = &t[1]
		}
	}

	if offset, ok := q["offset"]; ok {
		d.Offset = &offset[0]
	}

	if limit, ok := q["limit"]; ok {
		d.Limit = &limit[0]
	}

	return d
}

// ---

func (h *Handler) handleListOfTables(w http.ResponseWriter, r *http.Request) {
	// TODO: Check if the HTTP method is 'GET'

	tables := []string{}

	// collect table names
	for _, table := range h.Tables {
		tables = append(tables, table.Name)
	}

	t := map[string]interface{}{"tables": tables} // intermediate view
	c := Response{"", t}                          // no error
	j, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(j)
}

// ---

// primary router
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		h.handleListOfTables(w, r)
		return
	} else {
		// custom logic inside
	}
}
