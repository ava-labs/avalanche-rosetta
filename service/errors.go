package service

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

var (
	// General errors
	errNotImplemented     = makeError(1, "Endpoint is not implemented", false)
	errNotSupported       = makeError(2, "Endpoint is not supported", false)
	errUnavailableOffline = makeError(3, "Endpoint is not available offline", false)
	errInternalError      = makeError(4, "Internal server error", true)
	errInvalidInput       = makeError(5, "Invalid input", false)
	errClientError        = makeError(6, "Client error", true)

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
)

func errorList() []*types.Error {
	return []*types.Error{
		errNotImplemented,
		errNotSupported,
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
}

func makeError(code int32, message string, retriable bool) *types.Error {
	return &types.Error{
		Code:      code,
		Message:   message,
		Retriable: retriable,
	}
}

func wrapError(rosettaErr *types.Error, message interface{}) *types.Error {
	if rosettaErr.Details == nil {
		rosettaErr.Details = map[string]interface{}{}
	}

	switch t := message.(type) {
	case error:
		rosettaErr.Details["error"] = t.Error()
	default:
		rosettaErr.Details["error"] = t
	}
	return rosettaErr
}
