package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type payload struct {
	Signature string `json:"s"`
	Version   int    `json:"v"`
	Payload   string `json:"p"`
}

type redirect struct {
	MandrillAccountID int      `json:"u"`
	Version           int      `json:"v"`
	URL               string   `json:"url"`
	ID                string   `json:"id"`
	URLIDs            []string `json:"url_ids"`
}

var safeHosts []string

func hostIsSafe(host string) bool {
	for _, safeHost := range safeHosts {
		if host == safeHost {
			return true
		}
	}

	return false
}

func clickHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	domain := vars["domain"]

	payloadParam := req.URL.Query().Get("p")

	if len(payloadParam) == 0 {
		http.Error(w, "Missing payload", http.StatusBadRequest)
		return
	}

	// Mandrill's payload doesn't have padding. Put it back.
	if mod := len(payloadParam) % 4; mod != 0 {
		payloadParam += strings.Repeat("=", 4-mod)
	}

	payloadBytes, err := base64.StdEncoding.DecodeString(payloadParam)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var p payload
	err = json.Unmarshal(payloadBytes, &p)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var data redirect
	err = json.Unmarshal([]byte(p.Payload), &data)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Since we can't verify Mandrill's signtuare, we need to check the URL is one of ours...
	u, err := url.Parse(data.URL)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if domain != u.Host {
		http.Error(w, "Domain in route does not match domain in payload", http.StatusBadRequest)
		return
	}

	if hostIsSafe(u.Host) {
		// http.Redirect(w, req, data.URL, http.StatusMovedPermanently)
		fmt.Fprintf(w, "Permanent redirect to: %s", data.URL)
	} else {
		http.Error(w, "URL in payload is not considered safe", http.StatusBadRequest)
	}
}

func main() {
	safeHosts = []string{
		"yearbook.com",
		"www.yearbook.com",
		"yearbookmachine.com",
		"www.yearbookmachine.com",
		"twitter.com",
		"www.twitter.com",
		"facebook.com",
		"www.facebook.com",
	}

	r := mux.NewRouter()
	r.HandleFunc("/track/click/{account_id}/{domain}", clickHandler)

	loggedRouter := handlers.LoggingHandler(os.Stdout, r)
	http.ListenAndServe(":8080", loggedRouter)
}
