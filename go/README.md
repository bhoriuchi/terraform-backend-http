# terraform-backend-http

An extendable HTTP backend framework for terraform written in GO

## Overview

There are two components to a backend.

1. The backend itself which provides generic `http.HandleFunc` functions for the various operations required by an http state backend. This allows compatibility with most popular http server frameworks and allows control over authentication, ssl, etc.

2. The store which is an interface that can be implemented to allow any storage type. This allows custom stores to be developed outside this repo.

## Example

```go
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
```