package service

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

var (
	// Errors lists all available error types
	Errors = []*types.Error{
		ErrNotReady,
		ErrNotImplemented,
		ErrNotSupported,
		ErrUnavailableOffline,
		ErrInternalError,
		ErrInvalidInput,
		ErrClientError,
		ErrBlockInvalidInput,
		ErrBlockNotFound,
		ErrCallInvalidMethod,
		ErrCallInvalidParams,
		ErrTransactionNotFound,
	}

	// General errors
	ErrNotReady            = makeError(1, "Node is not ready", true)
	ErrNotImplemented      = makeError(2, "Endpoint is not implemented", false)
	ErrNotSupported        = makeError(3, "Endpoint is not supported", false)
	ErrUnavailableOffline  = makeError(4, "Endpoint is not available offline", false)
	ErrInternalError       = makeError(5, "Internal server error", true)
	ErrInvalidInput        = makeError(6, "Invalid input", false)
	ErrClientError         = makeError(7, "Client error", true)
	ErrBlockInvalidInput   = makeError(8, "Block number or hash is required", false)
	ErrBlockNotFound       = makeError(9, "Block was not found", true)
	ErrCallInvalidMethod   = makeError(10, "Invalid call method", false)
	ErrCallInvalidParams   = makeError(11, "invalid call params", false)
	ErrTransactionNotFound = makeError(12, "Transaction was not found", true)
)

func makeError(code int32, message string, retriable bool) *types.Error {
	return &types.Error{
		Code:      code,
		Message:   message,
		Retriable: retriable,
		Details:   map[string]interface{}{},
	}
}

func WrapError(err *types.Error, message interface{}) *types.Error {
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
