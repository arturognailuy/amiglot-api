package service

import "fmt"

type Error struct {
	Status int
	Key    string
	Err    error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Key, e.Err)
	}
	return e.Key
}

func (e *Error) Unwrap() error {
	return e.Err
}
