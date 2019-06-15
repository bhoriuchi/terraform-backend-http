package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/viper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Defaults
const (
	DefaultLockCollectionName  = "terraform_locks"
	DefaultStateCollectionName = "terraform_states"
	DefaultConnectTimeout      = 60
	DefaultQueryTimeout        = 5
)

var trueValue = true

// Options MongoDB backend options
type Options struct {
	LockCollectionName  string
	StateCollectionName string
	Database            string
	URI                 string
	ConnectTimeout      int
	QueryTimeout        int
}

// NewStore creates a new MongoDB backend
func NewStore(opts *Options) *Store {
	if opts == nil {
		opts = &Options{}
	}
	backend := Store{
		lockCollectionName:  opts.LockCollectionName,
		stateCollectionName: opts.StateCollectionName,
		database:            opts.Database,
		uri:                 opts.URI,
		connectTimeout:      opts.ConnectTimeout,
		queryTimeout:        opts.QueryTimeout,
	}

	if backend.lockCollectionName == "" {
		backend.lockCollectionName = DefaultLockCollectionName
	}

	if backend.stateCollectionName == "" {
		backend.stateCollectionName = DefaultStateCollectionName
	}

	if backend.connectTimeout == 0 {
		backend.connectTimeout = DefaultConnectTimeout
	}

	if backend.queryTimeout == 0 {
		backend.queryTimeout = DefaultQueryTimeout
	}

	return &backend
}

// Store MongoDB store
type Store struct {
	client              *mongo.Client
	lockCollectionName  string
	stateCollectionName string
	uri                 string
	database            string
	connectTimeout      int
	queryTimeout        int
}

// Init initializes the backend
func (c *Store) Init() error {
	c.connectTimeout = viper.GetInt("backend.connect_timeout")
	c.queryTimeout = viper.GetInt("backend.query_timeout")

	if c.uri = viper.GetString("backend.uri"); c.uri == "" {
		return fmt.Errorf("no uri specified in backend configuration")
	}

	if c.database = viper.GetString("backend.database"); c.database == "" {
		return fmt.Errorf("no database specified in backend configuration")
	}

	// connect
	clientOptions := options.Client().ApplyURI(c.uri)
	ctx, cancelFunc := newContext(&c.connectTimeout)
	defer cancelFunc()

	if client, err := mongo.Connect(ctx, clientOptions); err == nil {
		c.client = client
	} else {
		return err
	}

	// create indexes
	indexMap := map[string]string{
		c.lockCollectionName:  "ref",
		c.stateCollectionName: "ref",
	}

	for collection, field := range indexMap {
		if _, err := c.createUniqueIndex(collection, field); err != nil {
			return err
		}
	}

	return nil
}

// returns a collection
func (c *Store) collection(name string) *mongo.Collection {
	return c.client.Database(c.database).Collection(name)
}

// createUniqueIndex creates a unique index
func (c *Store) createUniqueIndex(collection, field string) (string, error) {
	ctx, cancelFunc := newContext(&c.queryTimeout)
	defer cancelFunc()

	return c.collection(collection).Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    map[string]int{field: 1},
		Options: &options.IndexOptions{Unique: &trueValue},
	})
}

// creates a new context optionally with timeout
func newContext(timeout ...*int) (context.Context, context.CancelFunc) {
	if len(timeout) > 0 {
		to := timeout[0]
		if to != nil && *to > 0 {
			return context.WithTimeout(
				context.Background(),
				time.Duration(*to)*time.Second,
			)
		}
	}
	return context.Background(), func() {}
}
