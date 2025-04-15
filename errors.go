package tfe

import "errors"

// Library errors that usually indicate a bug in the implementation of go-tfe
var (
	ErrItemsMustBeSlice = errors.New(`model field "Items" must be a slice`) // ErrItemsMustBeSlice is returned when an API response attribute called Items is not a slice

	ErrInvalidRequestBody = errors.New("go-tfe bug: DELETE/PATCH/POST body must be nil, ptr, or ptr slice") // ErrInvalidRequestBody is returned when a request body for DELETE/PATCH/POST is not a reference type

	ErrInvalidStructFormat = errors.New("go-tfe bug: struct can't use both json and jsonapi attributes") // ErrInvalidStructFormat is returned when a mix of json and jsonapi tagged fields are used in the same struct
)

var (
	ErrInvalidIncludeValue = errors.New(`invalid value for "include" field`)

	// ErrUnauthorized is returned when receiving a 401.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrResourceNotFound is returned when receiving a 404.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrMissingDirectory is returned when the path does not have an existing directory.
	ErrMissingDirectory = errors.New("path needs to be an existing directory")
)
