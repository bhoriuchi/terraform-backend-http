package mongodb

import (
	"github.com/bhoriuchi/terraform-backend-http/go/store"
	"github.com/bhoriuchi/terraform-backend-http/go/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetState gets the state
func (c *Store) GetState(ref string) (map[string]interface{}, bool, error) {
	ctx, cancelFunc := newContext(&c.queryTimeout)
	defer cancelFunc()

	result := c.collection(c.stateCollectionName).FindOne(ctx, bson.M{"ref": ref})
	if err := result.Err(); err != nil {
		return nil, false, err
	}

	var state types.StateDocument
	if err := result.Decode(&state); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, false, store.ErrNotFound
		}
		return nil, false, err
	}

	return state.State, state.Encrypted, nil
}

// PutState puts the state
func (c *Store) PutState(ref string, state, metadata map[string]interface{}, encrypted bool) error {
	ctx, cancelFunc := newContext(&c.queryTimeout)
	defer cancelFunc()

	document := types.StateDocument{
		Ref:       ref,
		State:     state,
		Encrypted: encrypted,
		Metadata:  metadata,
	}
	opts := options.FindOneAndReplaceOptions{Upsert: &trueValue}
	result := c.collection(c.stateCollectionName).FindOneAndReplace(ctx, bson.M{"ref": ref}, document, &opts)
	if err := result.Err(); err != nil {
		return err
	}

	return nil
}

// DeleteState deletes a state
func (c *Store) DeleteState(ref string) error {
	ctx, cancelFunc := newContext(&c.queryTimeout)
	defer cancelFunc()

	result := c.collection(c.stateCollectionName).FindOneAndDelete(ctx, bson.M{"ref": ref})
	if err := result.Err(); err != nil {
		return err
	}

	return nil
}
