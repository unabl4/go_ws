package main

import (
	"database/sql"
	"fmt"
	"net/http"
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
	return &h, nil    // at the moment, not entirely sure what the error is for
}

// ===

func (h *Handler) Initialize() error {
	var n string	// table name
	q, err := h.DB.Query("SHOW TABLES") // and not 'QueryRow'
	if err != nil {
		return err
	}
	 
	defer h.DB.Close()
	for q.Next() {	// loop through rows
		err := q.Scan(&n)
		if err != nil {
			return err
		}
	
		t := Table{n}
		h.Tables = append(h.Tables, t)
	}

	return nil	// no errors
}

// ---

// primary router
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(h)
}