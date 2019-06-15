package backend

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	gocrypto "github.com/bhoriuchi/go-crypto"
	"github.com/bhoriuchi/terraform-backend-http/go/store"
	"github.com/bhoriuchi/terraform-backend-http/go/types"
	"github.com/go-chi/render"
)

// Options backend options
type Options struct {
	EncryptionKey  interface{}
	Logger         func(level, message string, err error)
	GetRefFunc     interface{}
	GetEncryptFunc interface{}
}

// NewBackend creates a new backend
func NewBackend(store store.Store, opts ...*Options) *Backend {
	backend := Backend{
		initialized: false,
		store:       store,
	}

	if len(opts) > 0 {
		backend.options = opts[0]
	}
	if backend.options == nil {
		backend.options = &Options{}
	}

	return &backend
}

// Backend a terraform http backend
type Backend struct {
	initialized bool
	store       store.Store
	options     *Options
}

// Init initializes the backend
func (c *Backend) Init() error {
	if !c.initialized {
		return c.store.Init()
	}
	return nil
}

// gets the encryption key
func (c *Backend) getEncryptionKey() []byte {
	switch key := c.options.EncryptionKey; key.(type) {
	case []byte:
		return key.([]byte)
	case func() []byte:
		return key.(func() []byte)()
	}
	return nil
}

// gets the state ref
func (c *Backend) getRef(r *http.Request) string {
	switch refFunc := c.options.GetRefFunc; refFunc.(type) {
	case func(r *http.Request) string:
		return refFunc.(func(r *http.Request) string)(r)
	}
	return r.URL.Query().Get("ref")
}

// gets the encrypt state setting
func (c *Backend) getEncrypt(r *http.Request) bool {
	switch encFunc := c.options.GetRefFunc; encFunc.(type) {
	case func(r *http.Request) bool:
		return encFunc.(func(r *http.Request) bool)(r)
	}

	encrypt, err := strconv.ParseBool(r.URL.Query().Get("encrypt"))
	if err != nil {
		return false
	}
	return encrypt
}

// decrypts the encrypted state
func (c *Backend) decryptState(encryptedState interface{}) (map[string]interface{}, error) {
	key := c.getEncryptionKey()
	if key == nil || len(key) == 0 {
		return nil, fmt.Errorf("failed to get backend encryption key")
	}

	s := types.EncryptedState{}
	if err := toInterface(encryptedState, &s); err != nil {
		return nil, err
	}

	data, err := base64.StdEncoding.DecodeString(s.EncryptedData)
	if err != nil {
		return nil, err
	}

	decryptedData, err := gocrypto.Decrypt(key, data)
	if err != nil {
		return nil, err
	}

	var state map[string]interface{}
	if err := json.Unmarshal(decryptedData, &state); err != nil {
		return nil, err
	}

	return state, nil
}

// encrypts the state
func (c *Backend) encryptState(state interface{}) (map[string]interface{}, error) {
	key := c.getEncryptionKey()
	if key == nil || len(key) == 0 {
		return nil, fmt.Errorf("failed to get backend encryption key")
	}

	j, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}

	encryptedData, err := gocrypto.Encrypt(key, j)
	if err != nil {
		return nil, err
	}

	var encryptedState map[string]interface{}
	s := types.EncryptedState{
		EncryptedData: base64.StdEncoding.EncodeToString(encryptedData),
	}

	if err := toInterface(s, &encryptedState); err != nil {
		return nil, err
	}

	return encryptedState, nil
}

// determines if the state can be locked
func (c *Backend) canLock(w http.ResponseWriter, r *http.Request, ref, id string) bool {
	lock, err := c.store.GetLock(ref)
	if err != nil {
		if err == store.ErrNotFound {
			return true
		}

		c.options.Logger(
			"error",
			fmt.Sprintf("failed to get lock from state store for ref: %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return false
	}

	if lock.ID == id {
		return true
	}

	c.options.Logger(
		"debug",
		fmt.Sprintf("terraform state locked by another process for ref: %s", ref),
		nil,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusLocked)
	json.NewEncoder(w).Encode(lock)
	return false
}

// HandleGetState gets the state requested
func (c *Backend) HandleGetState(w http.ResponseWriter, r *http.Request) {
	ref := c.getRef(r)

	if err := c.Init(); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("failed to initialize terraform state backend for ref: %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.options.Logger(
		"debug",
		fmt.Sprintf("getting terraform state for ref: %s", ref),
		nil,
	)
	// get the state
	state, encrypted, err := c.store.GetState(ref)
	if err != nil {
		if err == store.ErrNotFound {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		c.options.Logger(
			"error",
			fmt.Sprintf("failed to get terraform state for ref: %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// decrypt
	if encrypted {
		decryptedState, err := c.decryptState(state)
		if err != nil {
			c.options.Logger(
				"error",
				fmt.Sprintf("failed decrypt terraform state for ref: %s", ref),
				err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		state = decryptedState
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(state)
}

// HandleLockState locks the state
func (c *Backend) HandleLockState(w http.ResponseWriter, r *http.Request) {
	ref := c.getRef(r)

	if err := c.Init(); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("failed to initialize terraform state backend for ref: %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.options.Logger(
		"debug",
		fmt.Sprintf("locking terraform state for ref %s", ref),
		nil,
	)

	// decode body
	var lock types.Lock
	if err := render.DecodeJSON(r.Body, &lock); err != nil {
		c.options.Logger(
			"debug",
			fmt.Sprintf("error decoding LOCK request body for ref %s", ref),
			nil,
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// check if state can be locked
	if !c.canLock(w, r, ref, lock.ID) {
		return
	}

	// attempt to put the lock
	if err := c.store.PutLock(ref, lock); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("failed to set lock for ref %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleUnlockState unlocks the state
func (c *Backend) HandleUnlockState(w http.ResponseWriter, r *http.Request) {
	ref := c.getRef(r)

	if err := c.Init(); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("failed to initialize terraform state backend for ref: %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.options.Logger(
		"debug",
		fmt.Sprintf("unlocking terraform state for ref %s", ref),
		nil,
	)

	// decode body
	var lock types.Lock
	if err := render.DecodeJSON(r.Body, &lock); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("error decoding UNLOCK request body for ref %s", ref),
			err,
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// check if state can be locked
	if !c.canLock(w, r, ref, lock.ID) {
		return
	}

	// attempt to delete the lock
	if err := c.store.DeleteLock(ref); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("failed to delete lock for ref %s", ref),
			err,
		)

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleUpdateState updates the state
func (c *Backend) HandleUpdateState(w http.ResponseWriter, r *http.Request) {
	ref := c.getRef(r)
	encrypt := c.getEncrypt(r)
	id := r.URL.Query().Get("ID")

	if err := c.Init(); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("failed to initialize terraform state backend for ref: %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.options.Logger(
		"debug",
		fmt.Sprintf("setting terraform state for ref %s", ref),
		nil,
	)

	// decode body
	var state map[string]interface{}
	if err := render.DecodeJSON(r.Body, &state); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("error decoding request body for ref %s", ref),
			err,
		)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !c.canLock(w, r, ref, id) {
		return
	}

	if encrypt {
		encryptedState, err := c.encryptState(state)
		if err != nil {
			c.options.Logger(
				"error",
				fmt.Sprintf("failed encrypt terraform state for ref: %s", ref),
				err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		state = encryptedState
	}

	// set the state on the backend
	if err := c.store.PutState(ref, state, encrypt); err != nil {
		c.options.Logger(
			"debug",
			fmt.Sprintf("error updating terraform state for ref %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandleDeleteState deletes the state
func (c *Backend) HandleDeleteState(w http.ResponseWriter, r *http.Request) {
	ref := c.getRef(r)
	id := r.URL.Query().Get("ID")

	if err := c.Init(); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("failed to initialize terraform state backend for ref: %s", ref),
			err,
		)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.options.Logger(
		"debug",
		fmt.Sprintf("deleting terraform state for ref %s", ref),
		nil,
	)

	if !c.canLock(w, r, ref, id) {
		return
	}

	if err := c.store.DeleteState(ref); err != nil {
		c.options.Logger(
			"error",
			fmt.Sprintf("error deleting terraform state for ref %s", ref),
			err,
		)
		if err == store.ErrNotFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// simple interface
func toInterface(input, output interface{}) error {
	j, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(j, output)
}
