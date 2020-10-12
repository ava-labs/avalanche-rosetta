package service

import (
	"github.com/coinbase/rosetta-sdk-go/types"
)

var (
	// General errors
	errNotImplemented = makeError(1, "Endpoint is not implemented", false)
	errNotSupported   = makeError(2, "Endpoint is not supported", false)
	errInternalError  = makeError(3, "Internal server error", true)
	errInvalidInput   = makeError(4, "Invalid input", false)

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
	errConstructionInvalidTx = makeError(300, "Invalid transaction data", false)
	errConstructionSubmit    = makeError(301, "Transaction submission failed", true)
)

func allErrors() []*types.Error {
	return []*types.Error{
		errInternalError,

		errStatusBlockFetchFailed,
		errStatusBlockNotFound,
		errStatusPeersFailed,

		errBlockInvalidInput,
		errBlockFetchFailed,
		errBlockNotFound,
	}
}

func makeError(code int32, message string, retriable bool) *types.Error {
	return &types.Error{
		Code:      code,
		Message:   message,
		Retriable: retriable,
	}
}

func errorWithInfo(rosettaErr *types.Error, err error) *types.Error {
	rosettaErr.Details["error"] = err.Error()
	return rosettaErr
}
