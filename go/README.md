# terraform-backend-http

An extendable HTTP backend framework for terraform written in GO

[![Documentation](https://godoc.org/github.com/bhoriuchi/terraform-backend-http/go?status.svg)](https://godoc.org/github.com/bhoriuchi/terraform-backend-http/go)


## Features

* Optional state encryption with AES-256-GCM
* Custom state metadata extraction
* Extensible store

## Overview

There are two components to a backend.

1. The backend itself which provides generic `http.HandleFunc` functions for the various operations required by an http state backend. This allows compatibility with most popular http server frameworks and allows control over authentication, ssl, etc.

2. The store which is an interface that can be implemented to allow any storage type. This allows custom stores to be developed outside this repo.

States are identified by a `ref` value. By default this value is obtained from the query string, but a custom `GetRefFunc` can be provided in the backend options to get the ref from another source (i.e. a route path variable)

Encryption can be enabled for a state by setting `encrypt=true` and providing an encryption key to the backend options. By default this value is obtained from the query string, but a custom `GetEncryptFunc` can be provided in the backend options to get the encrypt value from another source

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
	tfbackend := backend.NewBackend(store, &backend.Options{
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
	if err := tfbackend.Init(); err != nil {
		log.Fatal(err)
	}

	// add handlers
	http.HandleFunc("/backend", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "LOCK":
			tfbackend.HandleLockState(w, r)
		case "UNLOCK":
			tfbackend.HandleUnlockState(w, r)
		case http.MethodGet:
			tfbackend.HandleGetState(w, r)
		case http.MethodPost:
			tfbackend.HandleUpdateState(w, r)
		case http.MethodDelete:
			tfbackend.HandleDeleteState(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	log.Println("Starting test server on :3000")
  log.Fatal(http.ListenAndServe(":3000", nil))
}
```

## Encryption

Encryption can be enabled by providing an `EncryptionKey` to the backend options. This can either by a `[]byte` or a func that returns a `[]byte` containing the encryption key. As well as adding an `encrypt=true` query string to the backend URL. Additionally you can provide your own ref and encrypt getters in the backend options with `GetRefFunc` and `GetEncryptFunc`

```hcl
terraform {
  backend "http" {
    address = "http://localhost:3000/backend?ref=foo&encrypt=true"
    lock_address = "http://localhost:3000/backend?ref=foo&encrypt=true"
    unlock_address = "http://localhost:3000/backend?ref=foo&encrypt=true"
  }
}

resource "local_file" "testfile" {
  content = "foobar"
  filename = "${path.module}/test.json"
}
```

## Logging

A logging function can be provided to the backend options with `Logger` that will be called on informational and error events

## Extending

To add a custom http backend store, simply implement the `Store` interface

```go
type Store interface {
	Init() error

	// state
	GetState(ref string) (state map[string]interface{}, encrypted bool, err error)
	PutState(ref string, state, metadata map[string]interface{}, encrypted bool) error
	DeleteState(ref string) error

	// lock
	GetLock(ref string) (lock *types.Lock, err error)
	PutLock(ref string, lock types.Lock) error
	DeleteLock(ref string) error
}
```

Also note that it is important the `GetState` method return the `store.ErrNotFound` error when a state does not exist. This error is used to identify when an `http.StatusNoContent` response should be sent