package mongodb

import (
	"github.com/bhoriuchi/terraform-backend-http/go/store"
	"github.com/bhoriuchi/terraform-backend-http/go/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetLock gets the lock
func (c *Store) GetLock(ref string) (*types.Lock, error) {
	ctx, cancelFunc := newContext(&c.queryTimeout)
	defer cancelFunc()

	result := c.collection(c.lockCollectionName).FindOne(ctx, bson.M{"ref": ref}, &options.FindOneOptions{})
	if err := result.Err(); err != nil {
		return nil, err
	}

	var lock types.LockDocument
	if err := result.Decode(&lock); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	return &lock.Lock, nil
}

// PutLock puts the lock
func (c *Store) PutLock(ref string, lock types.Lock) error {
	ctx, cancelFunc := newContext(&c.queryTimeout)
	defer cancelFunc()

	document := types.LockDocument{
		Ref:  ref,
		Lock: lock,
	}
	opts := options.FindOneAndReplaceOptions{Upsert: &trueValue}
	result := c.collection(c.lockCollectionName).FindOneAndReplace(ctx, bson.M{"ref": ref}, document, &opts)
	if err := result.Err(); err != nil {
		return err
	}

	return nil
}

// DeleteLock deletes a lock
func (c *Store) DeleteLock(ref string) error {
	ctx, cancelFunc := newContext(&c.queryTimeout)
	defer cancelFunc()

	result := c.collection(c.lockCollectionName).FindOneAndDelete(ctx, bson.M{"ref": ref})
	if err := result.Err(); err != nil {
		return err
	}

	return nil
}
