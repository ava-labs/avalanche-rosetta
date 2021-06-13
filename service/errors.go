package service

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

var (
	// Errors lists all available error types
	Errors = []*types.Error{
		errNotImplemented,
		errNotSupported,
		errNotReady,
		errUnavailableOffline,
		errInternalError,
		errInvalidInput,
		errClientError,
		errBlockInvalidInput,
		errBlockNotFound,
		errCallInvalidMethod,
	}

	// General errors
	errNotReady           = makeError(1, "Node is not ready", true)                  //nolint:gomnd
	errNotImplemented     = makeError(2, "Endpoint is not implemented", false)       //nolint:gomnd
	errNotSupported       = makeError(3, "Endpoint is not supported", false)         //nolint:gomnd
	errUnavailableOffline = makeError(4, "Endpoint is not available offline", false) //nolint:gomnd
	errInternalError      = makeError(5, "Internal server error", true)              //nolint:gomnd
	errInvalidInput       = makeError(6, "Invalid input", false)                     //nolint:gomnd
	errClientError        = makeError(7, "Client error", true)                       //nolint:gomnd
	errBlockInvalidInput  = makeError(8, "Block number or hash is required", false)  //nolint:gomnd
	errBlockNotFound      = makeError(9, "Block was not found", true)                //nolint:gomnd
	errCallInvalidMethod  = makeError(10, "Invalid call method", false)              //nolint:gomnd
	errCallInvalidParams  = makeError(11, "invalid call params", false)              //nolint:gomnd
)

func makeError(code int32, message string, retriable bool) *types.Error {
	return &types.Error{
		Code:      code,
		Message:   message,
		Retriable: retriable,
		Details:   map[string]interface{}{},
	}
}

func wrapError(err *types.Error, message interface{}) *types.Error {
	newErr := makeError(err.Code, err.Message, err.Retriable)

	if err.Description != nil {
		newErr.Description = err.Description
	}

	for k, v := range err.Details {
		newErr.Details[k] = v
	}

	switch t := message.(type) {
	case error:
		newErr.Details["error"] = t.Error()
	default:
		newErr.Details["error"] = t
	}

	return newErr
}
