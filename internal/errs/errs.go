package errs

import "errors"

var (
	ErrOperatorNotFound      = errors.New("operator not found")
	ErrOperatorAlreadyExists = errors.New("operator profile already exists")
)
