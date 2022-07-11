package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type coasterHandlers struct {
	sync.Mutex
	store map[string]Coaster
}

func (h *coasterHandlers) coasters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.get(w, r)
		return
	case "POST":
		h.post(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method Not Allowed"))
	}
}

// GET
func (h *coasterHandlers) get(w http.ResponseWriter, r *http.Request) {
	coasters := make([]Coaster, len(h.store))

	i := 0
	for _, value := range h.store {
		coasters[i] = value
		i++
	}

	jsonBytes, err := json.Marshal(coasters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

// GET /coaster/id
func (h *coasterHandlers) getCoaster(w http.ResponseWriter, r *http.Request) {
	/*
		fmt.Print("print url: ", r.URL.String())
		from postman: 		localhost:8080/coasters/1657565645978787000
		r.URL.String(): 				  /coasters/1657565645978787000
	*/
	parts := strings.Split(r.URL.String(), "/")

	// if uri is not long enough return error 'not found'
	if len(parts) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// redirection scenario | getRandomCoaster
	if parts[2] == "random" {
		h.getRandomCoaster(w, r)
		return
	}

	//fetch the record from store by key
	h.Lock()
	coaster, exists := h.store[parts[2]]
	h.Unlock()

	// record not found by key, return 'not found'
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// json encode the record
	jsonEncodedRecord, err := json.Marshal(coaster)
	// error occured while converting to json, return 'internal server error'
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}

	// set content type to application/json as we are returning json response
	w.Header().Add("content-type", "application/json")
	// set status OK
	w.WriteHeader(http.StatusOK)
	// write
	w.Write(jsonEncodedRecord)
}

// GET /coaster/random | redirection usecase
func (h *coasterHandlers) getRandomCoaster(w http.ResponseWriter, r *http.Request) {
	coastersIds := make([]string, len(h.store))

	h.Lock()
	i := 0
	for value := range h.store {
		coastersIds[i] = value
		i++
	}
	defer h.Unlock()

	var target string
	if len(coastersIds) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if len(coastersIds) == 1 {
		target = coastersIds[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		target = coastersIds[rand.Intn(len(coastersIds))]
	}

	w.Header().Add("location", fmt.Sprintf("/coasters/%s", target))
	w.WriteHeader(http.StatusFound)

}

// POST single record

func (h *coasterHandlers) post(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	// general error check
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	ct := r.Header.Get("content-type")

	// check payload media type
	if ct != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content-type 'application/json', but got '%s'", ct)))
		return
	}

	//unmarshal json payload (accepted as byte array)
	var coaster Coaster
	err = json.Unmarshal(bodyBytes, &coaster)
	// error while unmarshalling payload, bad request
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// generate random number ID using time
	coaster.ID = fmt.Sprintf("%d", time.Now().UnixNano())

	// lock the table/store and insert
	h.Lock()
	h.store[coaster.ID] = coaster
	defer h.Unlock()

}

// constructor
func newCoasterHandlers() *coasterHandlers {
	return &coasterHandlers{
		store: map[string]Coaster{
			"id1": {
				Name:         "Furry 325",
				Manufacturer: "B+M",
				Height:       99,
				InPark:       "CaroWinds",
				ID:           "id1",
			},
		},
	}
}

// response type struct, you will see this in postman response
type Coaster struct {
	Name         string `json:"name"`
	Manufacturer string `json:"manufacturer"`
	ID           string `json:"id"`
	InPark       string `json:"inPark"`
	Height       int    `json:"height"`
}

type adminPortal struct {
	password string
}

func newAdminPortal() *adminPortal {
	pass := os.Getenv("ADMIN_PASSWORD")
	if pass == "" {
		panic("required env var ADMIN_PASSWORD")
	}

	return &adminPortal{
		password: pass,
	}
}

// Basic Auth, pass username and password from postman Authorization-> BasicAuth
func (a *adminPortal) adminHandler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()

	if !ok || user != "admin" || pass != a.password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized"))
		return
	}
	w.Write([]byte("<html><h1>Super Secret Admin Portal</h1></html>"))

}

func main() {
	/*
		start the program using following command
		ADMIN_PASSWORD=secret go run main.go
	*/

	admin := newAdminPortal()
	coasterHandlers := newCoasterHandlers()
	http.HandleFunc("/coasters", coasterHandlers.coasters)
	//everything that has `/`` after `/coasters`` in url is routed to .getCoaster
	http.HandleFunc("/coasters/", coasterHandlers.getCoaster)
	http.HandleFunc("/coasters/random", coasterHandlers.getRandomCoaster)
	http.HandleFunc("/admin", admin.adminHandler)
	fmt.Println("Hello World!")

	//start server and listen on tcp 8080
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		panic(err)
	}
}
