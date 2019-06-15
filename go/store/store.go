package store

import (
	"errors"

	"github.com/bhoriuchi/terraform-backend-http/go/types"
)

// ErrNotFound item not found
var ErrNotFound = errors.New("resource not found")

// Store store interface
type Store interface {
	Init() error

	// state
	GetState(ref string) (state interface{}, encrypted bool, err error)
	PutState(ref string, state interface{}, encrypted bool) error
	DeleteState(ref string) error

	// lock
	GetLock(ref string) (lock *types.Lock, err error)
	PutLock(ref string, lock types.Lock) error
	DeleteLock(ref string) error
}
