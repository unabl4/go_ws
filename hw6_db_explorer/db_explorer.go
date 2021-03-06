package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

const MAX_LIMIT = int(1e3) // huge number

type Handler struct {
	DB *sql.DB
	// ---
	Tables map[string]Table // []Tables
}

type Table struct {
	Name   string
	Fields []Field
}

type Field struct {
	Name string
	Type string
	// ---
	IsPrimary       bool
	IsAutoIncrement bool
	IsNullable      bool
}

// ===

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	h := Handler{} // new instance
	h.DB = db
	h.Tables = make(map[string]Table) // required
	err := h.Initialize()
	if err != nil {
		return nil, err
	}
	return &h, nil // at the moment, not entirely sure what the error is for
}

// ===

func getTableFields(db *sql.DB, tableName string) ([]Field, error) {
	// sql -> go type converter routine
	t := func(i string) string {
		if strings.HasPrefix(i, "int") {
			return "int"
		} else if strings.HasPrefix(i, "varchar") || i == "text" {
			return "string"
		} else {
			panic("unsupported field type")
		}
	} // end of t

	r, err := db.Query(fmt.Sprintf("SHOW FULL COLUMNS FROM %s", tableName)) // placeholder
	if err != nil {
		return nil, err
	}

	defer r.Close() // close the statement

	var fields []Field
	var w interface{}                                            // waste var (= value ignore)
	var fieldType, isNullable, isPrimary, isAutoIncrement string // aux
	for r.Next() {
		f := Field{}
		err := r.Scan(&f.Name, &fieldType, &w, &isNullable, &isPrimary, &w, &isAutoIncrement, &w, &w)
		if err != nil {
			return nil, err
		}

		f.Type = t(fieldType)

		if isNullable == "YES" {
			f.IsNullable = true
		}

		// primary key is ignored during the insertion and CANNOT be updated
		if isPrimary == "PRI" {
			f.IsPrimary = true
		}

		if isAutoIncrement == "auto_increment" {
			f.IsAutoIncrement = true
		}

		fields = append(fields, f)
	}

	return fields, nil
}

func (h *Handler) Initialize() error {
	var n string                        // table name
	q, err := h.DB.Query("SHOW TABLES") // and not 'QueryRow'
	if err != nil {
		return err
	}

	defer q.Close() // close the statement
	for q.Next() {  // loop through rows
		err := q.Scan(&n)
		if err != nil {
			return err
		}

		t := Table{}
		t.Name = n
		// tables to be defined later

		h.Tables[n] = t
	}

	// separate loop to get the fields
	for _, t := range h.Tables {
		fields, err := getTableFields(h.DB, t.Name) // ?
		if err != nil {
			return err
		}

		t.Fields = fields
		h.Tables[t.Name] = t
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

type DbQuery struct {
	Method    string
	TableName string // for what table
	// optional params
	RecordId *string // particular record we are after

	Offset int // defaults to zero
	Limit  int
}

// extract necessary params from the request url (table name, record id) path and query (limit, offset)
func parseDbQuery(r *http.Request) DbQuery {
	p := strings.Split(r.URL.Path, "/") // path components
	q := r.URL.Query()                  // query components
	d := DbQuery{}
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
		offsetInt, err := strconv.Atoi(offset[0])
		if err == nil { // string -> int conversion failed
			d.Offset = offsetInt // otherwise defaults to zero
		}

		// if the offset is set -> the limit MUST be set automatically
		d.Limit = MAX_LIMIT
	}

	if limit, ok := q["limit"]; ok {
		limitInt, err := strconv.Atoi(limit[0])
		if err != nil { // string -> int conversion failed
			limitInt = MAX_LIMIT
		}
		d.Limit = limitInt
	}

	// TODO: Validation? (GET, POST, PUT, DELETE)
	d.Method = r.Method
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

	sort.Strings(tables)
	t := map[string]interface{}{"tables": tables} // intermediate view
	c := Response{"", t}                          // no error
	j, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Write(j)
}

func (h *Handler) handleShow(w http.ResponseWriter, r *http.Request, q DbQuery) {
	t, ok := h.Tables[q.TableName] // check the table
	if !ok {
		// table not found -> 404
		c := Response{"unknown table", nil}
		j, err := json.Marshal(c)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusNotFound) // and not '200'
		w.Write(j)                         // the content
		return
	}

	// ---

	var endJson interface{} // map

	if q.RecordId != nil {
		// lookup for a particular record
		// ignore limit and offset (I guess)

		var primaryKey string
		for _, f := range t.Fields {
			if f.IsPrimary {
				primaryKey = f.Name
				break // found
			}
		}

		if primaryKey == "" {
			panic("Primary key not found")
		}

		queryStr := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", t.Name, primaryKey)
		row := h.DB.QueryRow(queryStr, q.RecordId)

		columns := make([]interface{}, len(t.Fields))
		columnPointers := make([]interface{}, len(columns)) // required
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		err := row.Scan(columnPointers...)
		if err != nil {
			// there's an error
			if err == sql.ErrNoRows {
				// special case -> no rows (records)

				c := Response{"record not found", nil}
				j, err := json.Marshal(c)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}

				w.WriteHeader(http.StatusNotFound) // and not '200'
				w.Write(j)                         // the content
				return
			} else {
				panic("unknown error occured") // shortcut to 00
			}
		}

		// record map composition
		record := make(map[string]interface{}) // new record
		for i, col := range t.Fields {
			colName := strings.ToLower(col.Name)
			value := columns[i]              // get the 'column' value
			bytes, ok := columns[i].([]byte) // important step, otherwise looks like base64 encoded (acts so as well)
			if ok {
				value = string(bytes)
			}

			record[colName] = value
		}

		endJson = map[string]interface{}{"record": record} // wrap
	} else {
		// show table records
		// (SELECT * FROM) -> json (<- map)
		// IDEA: pass map refs vector into Scan?

		placeholderVals := []interface{}{}
		queryStr := fmt.Sprintf("SELECT * FROM %s", t.Name) // placeholder cannot be used for table or column names (google)

		if q.Offset != 0 || q.Limit != 0 {
			queryStr += " LIMIT ?,?" // <offset, limit>
			placeholderVals = append(placeholderVals, q.Offset, q.Limit)
		}

		rows, err := h.DB.Query(queryStr, placeholderVals...)
		if err != nil {
			panic(err) // shortcut to 500 (recovery)
		}

		var g []map[string]interface{}
		defer rows.Close() // close statement
		for rows.Next() {
			columns := make([]interface{}, len(t.Fields))
			columnPointers := make([]interface{}, len(columns)) // required
			for i := range columns {
				columnPointers[i] = &columns[i]
			}

			err := rows.Scan(columnPointers...)
			if err != nil {
				panic(err) // shortcut to 500
			}

			// record map composition
			record := make(map[string]interface{}) // new record
			for i, col := range t.Fields {
				colName := strings.ToLower(col.Name)
				value := columns[i] // get the 'column' value

				bytes, ok := columns[i].([]byte) // important step, otherwise looks like base64 encoded (acts so as well)
				if ok {
					strValue := string(bytes)
					if col.Type == "int" { // special int case
						intValue, err := strconv.Atoi(strValue)
						if err != nil {
							panic("string -> int conversion error")
						}
						value = intValue
					} else {
						value = strValue
					}
				}

				record[colName] = value
			}

			g = append(g, record)
		}

		endJson = map[string]interface{}{"records": g} // wrap
	} // end of if

	// ---

	c := Response{"", endJson} // wrap in 'response'
	j, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return // ?
	}

	// 200,OK
	w.Write(j) // the content
} // end of handle show

// routine to generate an error message
func invalidField(field string, w http.ResponseWriter, r *http.Request) {
	errMsg := fmt.Sprintf("field %s have invalid type", field)
	c := Response{errMsg, nil} // wrap in 'response'
	j, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return // ?
	}

	w.WriteHeader(http.StatusBadRequest) // 400
	w.Write(j)                           // the content
}

// PUT -> add (incorrect, but still)
func (h *Handler) handleAdd(w http.ResponseWriter, r *http.Request, q DbQuery) {
	// cases:
	// 1) insertion into a non-existing table
	// 2) insertion with incorrect args
	// 3) invalid field TYPE value (from tests)

	t, tableExists := h.Tables[q.TableName]
	if q.RecordId != nil || !tableExists {
		// bad request: either it's a request for a particular record or the table does not exist
		http.Error(w, "bad request", http.StatusBadRequest)
		return // ?
	}

	jsonBody, err := ioutil.ReadAll(r.Body) // raw json body
	if err != nil {
		panic(err) // shortcut to 500
	}

	body := make(map[string]interface{})
	json.Unmarshal(jsonBody, &body) // check for errors

	// ---

	var primaryKey string
	placeholders := []string{}
	placeholderVals := []interface{}{}
	fieldNames := []string{}

	for _, f := range t.Fields {
		k := strings.ToLower(f.Name) // key
		fieldValue, valuePresent := body[k]

		if f.IsPrimary {
			primaryKey = f.Name
			continue
		}

		if !valuePresent {
			// value not present -> set default
			if f.IsNullable {
				fieldValue = nil
			} else {
				if f.Type == "string" {
					fieldValue = ""
				} else if f.Type == "int" {
					fieldValue = 0
				} else {
					panic("unknown type")
				}
			}
		} else {
			// some value is already set -> need to validate
			switch fieldValue.(type) {
			case float64: // weird part over here (dunno)
				if f.Type == "string" {
					invalidField(k, w, r)
					return
				}
			case string:
				if f.Type == "int" {
					invalidField(k, w, r)
					return
				}
			default:
				// hopefully will never happen
				invalidField(k, w, r)
				return
			}
		}

		fieldNames = append(fieldNames, f.Name)
		placeholders = append(placeholders, "?")
		placeholderVals = append(placeholderVals, fieldValue)
	}

	queryStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", t.Name, strings.Join(fieldNames, ","), strings.Join(placeholders, ","))
	result, err := h.DB.Exec(queryStr, placeholderVals...)

	id, err := result.LastInsertId()
	response := map[string]interface{}{primaryKey: id} // the main ID response
	c := Response{"", response}
	j, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return // ?
	}

	w.Write(j)
} // end of 'handleAdd'

// POST -> update (incorrect, but still)
func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request, q DbQuery) {
	t, tableExists := h.Tables[q.TableName]

	if q.RecordId == nil || !tableExists {
		// the request must be made for a particular record
		// TODO: what if it does not exist?
		http.Error(w, "bad request", http.StatusBadRequest)
		return // ?
	}

	jsonBody, err := ioutil.ReadAll(r.Body) // raw json body
	if err != nil {
		panic(err) // shortcut to 500
	}

	body := make(map[string]interface{})
	json.Unmarshal(jsonBody, &body) // check for errors

	// ---

	var primaryKey string
	placeholders := []string{}
	placeholderVals := []interface{}{}
	fieldNames := []string{}

	for _, f := range t.Fields {
		k := strings.ToLower(f.Name) // key
		fieldValue, valuePresent := body[k]

		if f.IsPrimary {
			if valuePresent {
				// uh-oh -> not good
				invalidField(k, w, r)
				return
			} else {
				primaryKey = f.Name
				continue
			}
		}

		// not present -> skip
		if !valuePresent {
			continue
		}

		switch fieldValue.(type) {
		case float64: // weird part over here (dunno)
			if f.Type == "string" {
				invalidField(k, w, r)
				return
			}
		case string:
			if f.Type == "int" {
				invalidField(k, w, r)
				return
			}
		case nil: // quite special case
			if !f.IsNullable {
				invalidField(k, w, r)
				return
			}
		default:
			// hopefully will never happen
			invalidField(k, w, r)
			return
		}

		fieldNames = append(fieldNames, fmt.Sprintf("%s = ?", f.Name))
		placeholders = append(placeholders, "?")
		placeholderVals = append(placeholderVals, fieldValue)
	}

	placeholderVals = append(placeholderVals, q.RecordId) // primary key

	queryStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", t.Name, strings.Join(fieldNames, ","), primaryKey)
	result, err := h.DB.Exec(queryStr, placeholderVals...)

	affected, err := result.RowsAffected()
	// TODO: handle err panic
	response := map[string]interface{}{"updated": affected}
	c := Response{"", response}
	j, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return // ?
	}

	w.Write(j)
} // end of 'handleUpdate'

// DELETE
func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request, q DbQuery) {
	t, tableExists := h.Tables[q.TableName]
	if q.RecordId == nil || !tableExists {
		// the request must be made for a particular record
		// TODO: what if it does not exist?
		http.Error(w, "bad request", http.StatusBadRequest)
		return // ?
	}

	// ---

	var primaryKey string

	for _, f := range t.Fields {
		if f.IsPrimary {
			primaryKey = f.Name
			break // found
		}
	}

	queryStr := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", t.Name, primaryKey)
	result, err := h.DB.Exec(queryStr, q.RecordId)

	affected, err := result.RowsAffected()
	// TODO: handle err panic
	response := map[string]interface{}{"deleted": affected}
	c := Response{"", response}
	j, err := json.Marshal(c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return // ?
	}

	w.Write(j)
} // end of 'handleDelete'

// ---

/*
* GET / - возвращает список все таблиц (которые мы можем использовать в дальнейших запросах)
* GET /$table?limit=5&offset=7 - возвращает список из 5 записей (limit) начиная с 7-й (offset) из таблицы $table. limit по-умолчанию 5, offset 0
* GET /$table/$id - возвращает информацию о самой записи или 404
* PUT /$table - создаёт новую запись, данный по записи в теле запроса (POST-параметры)
* POST /$table/$id - обновляет запись, данные приходят в теле запроса (POST-параметры)
* DELETE /$table/$id - удаляет запись */

// primary router
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// 500
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}()

	// ---
	// special case
	if r.URL.Path == "/" && r.Method == "GET" {
		h.handleListOfTables(w, r)
		return // stop
	}

	// the remaining part
	q := parseDbQuery(r)

	switch r.Method {
	case "GET":
		h.handleShow(w, r, q)
	case "POST":
		h.handleUpdate(w, r, q)
	case "PUT":
		h.handleAdd(w, r, q)
	case "DELETE":
		h.handleDelete(w, r, q)
	default:
		http.Error(w, "bad request", http.StatusBadRequest) // ?
	}
}
