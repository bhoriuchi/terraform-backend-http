# terraform-backend-http
An extendable HTTP backend framework for terraform

## Features

* Optional state encryption with AES-256-GCM
* Custom state metadata extraction
* Extensible store

## Overview

There are two components to a backend.

1. The backend itself which provides generic `http.HandleFunc` functions for the various operations required by an http state backend. This allows compatibility with most popular http server frameworks and allows control over authentication, ssl, etc.

2. The store which is an interface that can be implemented to allow any storage type. This allows custom stores to be developed outside this repo.

### Implementations

* Golang - [https://github.com/bhoriuchi/terraform-backend-http/tree/master/go](https://github.com/bhoriuchi/terraform-backend-http/tree/master/go)