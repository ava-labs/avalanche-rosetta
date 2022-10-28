package cchain

import "github.com/coinbase/rosetta-sdk-go/types"

// copied here from service. TODO cleanup

var (
	ErrUnavailableOffline = makeError(4, "Endpoint is not available offline", false)
	ErrNotImplemented     = makeError(2, "Endpoint is not implemented", false)
	ErrBlockInvalidInput  = makeError(8, "Block number or hash is required", false)
	ErrClientError        = makeError(7, "Client error", true)
	ErrInvalidInput       = makeError(6, "Invalid input", false)
	ErrInternalError      = makeError(5, "Internal server error", true)
	ErrBlockNotFound      = makeError(9, "Block was not found", true)
	ErrCallInvalidParams  = makeError(11, "invalid call params", false)
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
