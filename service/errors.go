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
		errStatusBlockFetchFailed,
		errStatusBlockNotFound,
		errStatusPeersFailed,
		errBlockInvalidInput,
		errBlockFetchFailed,
		errBlockNotFound,
		errConstructionSubmitFailed,
		errConstructionInvalidTx,
		errCallInvalidMethod,
	}

	// General errors
	errNotImplemented     = makeError(1, "Endpoint is not implemented", false)
	errNotSupported       = makeError(2, "Endpoint is not supported", false)
	errNotReady           = makeError(3, "Node is not ready", true)
	errUnavailableOffline = makeError(4, "Endpoint is not available offline", false)
	errInternalError      = makeError(5, "Internal server error", true)
	errInvalidInput       = makeError(6, "Invalid input", false)
	errClientError        = makeError(7, "Client error", true)

	// Network service errors
	errStatusBlockFetchFailed  = makeError(100, "Unable to fetch block", true)
	errStatusBlockNotFound     = makeError(101, "Latest block was not found", true)
	errStatusPeersFailed       = makeError(102, "Unable to fetch peers", true)
	errStatusNodeVersionFailed = makeError(103, "Unable to fetch node version", true)

	// Block service errors
	errBlockInvalidInput = makeError(200, "Block number or hash is required", false)
	errBlockFetchFailed  = makeError(201, "Unable to fetch block", true)
	errBlockNotFound     = makeError(202, "Block was not found", false)

	// Construction service errors
	errConstructionInvalidTx     = makeError(300, "Invalid transaction data", false)
	errConstructionInvalidPubkey = makeError(301, "Invalid public key data", false)
	errConstructionSubmitFailed  = makeError(302, "Transaction submission failed", true)

	// Call service errors
	errCallInvalidMethod = makeError(400, "Invalid call method", false)
	errCallInvalidParams = makeError(400, "invalid call params", false)
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
