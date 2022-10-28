package service

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	cBackend "github.com/ava-labs/avalanche-rosetta/backend/cchain"
)

// ConstructionBackend represents a backend that implements /construction family of apis for a subset of requests.
// Endpoint handlers in this file delegates requests to corresponding backends based on the request.
// Each backend implements a ShouldHandleRequest method to determine whether that backend should handle the given request.
//
// P-chain and C-chain atomic transaction logic are implemented in pchain.Backend and cchainatomictx.Backend respectively.
// Eventually, the C-chain non-atomic transaction logic implemented in this file should be extracted to its own backend as well.
type ConstructionBackend interface {
	// ShouldHandleRequest returns whether a given request should be handled by this backend
	ShouldHandleRequest(req interface{}) bool
	// ConstructionDerive implements /construction/derive endpoint for this backend
	ConstructionDerive(ctx context.Context, req *types.ConstructionDeriveRequest) (*types.ConstructionDeriveResponse, *types.Error)
	// ConstructionPreprocess implements /construction/preprocess endpoint for this backend
	ConstructionPreprocess(ctx context.Context, req *types.ConstructionPreprocessRequest) (*types.ConstructionPreprocessResponse, *types.Error)
	// ConstructionMetadata implements /construction/metadata endpoint for this backend
	ConstructionMetadata(ctx context.Context, req *types.ConstructionMetadataRequest) (*types.ConstructionMetadataResponse, *types.Error)
	// ConstructionPayloads implements /construction/payloads endpoint for this backend
	ConstructionPayloads(ctx context.Context, req *types.ConstructionPayloadsRequest) (*types.ConstructionPayloadsResponse, *types.Error)
	// ConstructionParse implements /construction/parse endpoint for this backend
	ConstructionParse(ctx context.Context, req *types.ConstructionParseRequest) (*types.ConstructionParseResponse, *types.Error)
	// ConstructionCombine implements /construction/combine endpoint for this backend
	ConstructionCombine(ctx context.Context, req *types.ConstructionCombineRequest) (*types.ConstructionCombineResponse, *types.Error)
	// ConstructionHash implements /construction/hash endpoint for this backend
	ConstructionHash(ctx context.Context, req *types.ConstructionHashRequest) (*types.TransactionIdentifierResponse, *types.Error)
	// ConstructionSubmit implements /construction/submit endpoint for this backend
	ConstructionSubmit(ctx context.Context, req *types.ConstructionSubmitRequest) (*types.TransactionIdentifierResponse, *types.Error)
}

// ConstructionService implements /construction/* endpoints
type ConstructionService struct {
	mode                  string
	cChainBackend         *cBackend.Backend
	cChainAtomicTxBackend ConstructionBackend
	pChainBackend         ConstructionBackend
}

// NewConstructionService returns a new construction servicer
func NewConstructionService(
	mode string,
	cChainBackend *cBackend.Backend,
	pChainBackend ConstructionBackend,
	cChainAtomicTxBackend ConstructionBackend,
) server.ConstructionAPIServicer {
	return &ConstructionService{
		mode:                  mode,
		cChainBackend:         cChainBackend,
		cChainAtomicTxBackend: cChainAtomicTxBackend,
		pChainBackend:         pChainBackend,
	}
}

// ConstructionMetadata implements /construction/metadata endpoint.
//
// Get any information required to construct a transaction for a specific network.
// Metadata returned here could be a recent hash to use, an account sequence number,
// or even arbitrary chain state. The request used when calling this endpoint
// is created by calling /construction/preprocess in an offline environment.
func (s ConstructionService) ConstructionMetadata(
	ctx context.Context,
	req *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	if s.mode == ModeOffline {
		return nil, ErrUnavailableOffline
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionMetadata(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionMetadata(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionMetadata(ctx, req)
}

// ConstructionHash implements /construction/hash endpoint.
//
// TransactionHash returns the network-specific transaction hash for a signed transaction.
func (s ConstructionService) ConstructionHash(
	ctx context.Context,
	req *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if len(req.SignedTransaction) == 0 {
		return nil, WrapError(ErrInvalidInput, "signed transaction value is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionHash(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionHash(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionHash(ctx, req)
}

// ConstructionCombine implements /construction/combine endpoint.
//
// Combine creates a network-specific transaction from an unsigned transaction
// and an array of provided signatures. The signed transaction returned from
// this method will be sent to the /construction/submit endpoint by the caller.
func (s ConstructionService) ConstructionCombine(
	ctx context.Context,
	req *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	if len(req.UnsignedTransaction) == 0 {
		return nil, WrapError(ErrInvalidInput, "transaction data is not provided")
	}
	if len(req.Signatures) == 0 {
		return nil, WrapError(ErrInvalidInput, "signature is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionCombine(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionCombine(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionCombine(ctx, req)
}

// ConstructionDerive implements /construction/derive endpoint.
//
// Derive returns the AccountIdentifier associated with a public key. Blockchains
// that require an on-chain action to create an account should not implement this method.
func (s ConstructionService) ConstructionDerive(
	ctx context.Context,
	req *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	if req.PublicKey == nil {
		return nil, WrapError(ErrInvalidInput, "public key is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionDerive(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionDerive(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionDerive(ctx, req)
}

// ConstructionParse implements /construction/parse endpoint
//
// Parse is called on both unsigned and signed transactions to understand the
// intent of the formulated transaction. This is run as a sanity check before signing
// (after /construction/payloads) and before broadcast (after /construction/combine).
func (s ConstructionService) ConstructionParse(
	ctx context.Context,
	req *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionParse(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionParse(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionParse(ctx, req)
}

// ConstructionPayloads implements /construction/payloads endpoint
//
// Payloads is called with an array of operations and the response from /construction/metadata.
// It returns an unsigned transaction blob and a collection of payloads that must
// be signed by particular AccountIdentifiers using a certain SignatureType.
// The array of operations provided in transaction construction often times can
// not specify all "effects" of a transaction (consider invoked transactions in Ethereum).
// However, they can deterministically specify the "intent" of the transaction,
// which is sufficient for construction. For this reason, parsing the corresponding
// transaction in the Data API (when it lands on chain) will contain a superset of
// whatever operations were provided during construction.
func (s ConstructionService) ConstructionPayloads(
	ctx context.Context,
	req *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionPayloads(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionPayloads(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionPayloads(ctx, req)
}

// ConstructionPreprocess implements /construction/preprocess endpoint.
//
// Preprocess is called prior to /construction/payloads to construct a request for
// any metadata that is needed for transaction construction given (i.e. account nonce).
func (s ConstructionService) ConstructionPreprocess(
	ctx context.Context,
	req *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionPreprocess(ctx, req)
	}
	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionPreprocess(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionPreprocess(ctx, req)
}

// ConstructionSubmit implements /construction/submit endpoint.
//
// Submit a pre-signed transaction to the node.
func (s ConstructionService) ConstructionSubmit(
	ctx context.Context,
	req *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.mode == ModeOffline {
		return nil, ErrUnavailableOffline
	}

	if len(req.SignedTransaction) == 0 {
		return nil, WrapError(ErrInvalidInput, "signed transaction value is not provided")
	}

	if s.pChainBackend.ShouldHandleRequest(req) {
		return s.pChainBackend.ConstructionSubmit(ctx, req)
	}

	if s.cChainAtomicTxBackend.ShouldHandleRequest(req) {
		return s.cChainAtomicTxBackend.ConstructionSubmit(ctx, req)
	}

	// TODO ABENEGIA: replace with ShouldHandleRequest
	// and return error if it's not even CChain block
	return s.cChainBackend.ConstructionSubmit(ctx, req)
}
