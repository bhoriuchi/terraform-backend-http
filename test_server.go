package main

import (
	"log"
	"net/http"

	backend "github.com/bhoriuchi/terraform-backend-http/go"
	"github.com/bhoriuchi/terraform-backend-http/go/store/mongodb"
)

func main() {
	// create a store
	store := mongodb.NewStore(&mongodb.Options{
		Database: "terraform",
		URI:      "mongodb://localhost:27017",
	})

	// create a backend
	backend := backend.NewBackend(store, &backend.Options{
		EncryptionKey: []byte("thisishardlysecure"),
		Logger: func(level, message string, err error) {
			if err != nil {
				log.Printf("%s: %s - %v", level, message, err)
			} else {
				log.Printf("%s: %s", level, message)
			}
		},
		GetMetadataFunc: func(state map[string]interface{}) map[string]interface{} {
			// fmt.Println(state)
			return map[string]interface{}{
				"test": "metadata",
			}
		},
	})
	if err := backend.Init(); err != nil {
		log.Fatal(err)
	}

	// add handlers
	http.HandleFunc("/backend", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "LOCK":
			backend.HandleLockState(w, r)
		case "UNLOCK":
			backend.HandleUnlockState(w, r)
		case http.MethodGet:
			backend.HandleGetState(w, r)
		case http.MethodPost:
			backend.HandleUpdateState(w, r)
		case http.MethodDelete:
			backend.HandleDeleteState(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Println("Starting test server on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
