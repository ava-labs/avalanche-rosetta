package service

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

var (
	// Errors lists all available error types
	Errors = []*types.Error{
		errNotReady,
		errNotImplemented,
		errNotSupported,
		errUnavailableOffline,
		errInternalError,
		errInvalidInput,
		errClientError,
		errBlockInvalidInput,
		errBlockNotFound,
		errCallInvalidMethod,
		errCallInvalidParams,
	}

	// General errors
	errNotReady           = makeError(1, "Node is not ready", true)
	errNotImplemented     = makeError(2, "Endpoint is not implemented", false)
	errNotSupported       = makeError(3, "Endpoint is not supported", false)
	errUnavailableOffline = makeError(4, "Endpoint is not available offline", false)
	errInternalError      = makeError(5, "Internal server error", true)
	errInvalidInput       = makeError(6, "Invalid input", false)
	errClientError        = makeError(7, "Client error", true)
	errBlockInvalidInput  = makeError(8, "Block number or hash is required", false)
	errBlockNotFound      = makeError(9, "Block was not found", true)
	errCallInvalidMethod  = makeError(10, "Invalid call method", false)
	errCallInvalidParams  = makeError(11, "invalid call params", false)
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
