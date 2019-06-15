package types

// EncryptedState encrypted state
type EncryptedState struct {
	EncryptedData string `json:"encrypted_data"`
}

// StateType a state
type StateType struct {
	Ref       string      `json:"ref"`
	Encrypted bool        `json:"encrypted"`
	State     interface{} `json:"state"`
}

// LockType a lock
type LockType struct {
	Ref  string `json:"ref"`
	Lock Lock   `json:"lock"`
}

// Lock a lock on state
type Lock struct {
	Created   string
	Path      string
	ID        string
	Operation string
	Info      string
	Who       string
	Version   string
}
