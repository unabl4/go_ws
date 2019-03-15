package main

import (
	"database/sql"
	"net/http"
)

type Handler struct {
	DbConn *sql.DB
}

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	h := Handler{db} // new instance
	return h, nil    // at the moment, not entirely sure what the error is for
}

// ---

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
}
