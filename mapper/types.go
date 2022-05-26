package mapper

import (
	"github.com/ava-labs/coreth/params"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	MainnetChainID = 43114
	MainnetAssetID = "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z"
	MainnetNetwork = "Mainnet"

	FujiChainID = 43113
	FujiAssetID = "U8iRqJoiJm8xZHAacmvYyZVwqQx6uDNtQeP3CQ6fcgQk3JqnK"
	FujiNetwork = "Fuji"

	ContractAddressMetadata  = "contractAddress"
	IndexTransferredMetadata = "indexTransferred"

	PChainNetworkIdentifier = "P"
	PChainIDAlias           = "P"

	OpCall          = "CALL"
	OpFee           = "FEE"
	OpCreate        = "CREATE"
	OpCreate2       = "CREATE2"
	OpSelfDestruct  = "SELFDESTRUCT"
	OpCallCode      = "CALLCODE"
	OpDelegateCall  = "DELEGATECALL"
	OpStaticCall    = "STATICCALL"
	OpDestruct      = "DESTRUCT"
	OpImport        = "IMPORT"
	OpExport        = "EXPORT"
	OpErc20Transfer = "ERC20_TRANSFER"
	OpErc20Mint     = "ERC20_MINT"
	OpErc20Burn     = "ERC20_BURN"

	OpErc721TransferSender  = "ERC721_SENDER"
	OpErc721TransferReceive = "ERC721_RECEIVE"
	OpErc721Mint            = "ERC721_MINT"
	OpErc721Burn            = "ERC721_BURN"

	StatusSuccess = "SUCCESS"
	StatusFailure = "FAILURE"
)

var (
	MainnetAP5Activation = params.AvalancheMainnetChainConfig.ApricotPhase5BlockTimestamp
	FujiAP5Activation    = params.AvalancheFujiChainConfig.ApricotPhase5BlockTimestamp

	StageBootstrap = &types.SyncStatus{
		Synced: types.Bool(false),
		Stage:  types.String("BOOTSTRAP"),
	}

	StageSynced = &types.SyncStatus{
		Synced: types.Bool(true),
		Stage:  types.String("SYNCED"),
	}

	AvaxCurrency = &types.Currency{
		Symbol:   "AVAX",
		Decimals: 18,
	}

	OperationStatuses = []*types.OperationStatus{
		{
			Status:     StatusSuccess,
			Successful: true,
		},
		{
			Status:     StatusFailure,
			Successful: false,
		},
	}

	OperationTypes = []string{
		OpFee,
		OpCall,
		OpCreate,
		OpCreate2,
		OpSelfDestruct,
		OpCallCode,
		OpDelegateCall,
		OpStaticCall,
		OpDestruct,
		OpImport,
		OpExport,
		OpErc20Burn,
		OpErc20Mint,
		OpErc20Transfer,
		OpErc721TransferReceive,
		OpErc721TransferSender,
		OpErc721Mint,
		OpErc721Burn,
	}

	CallMethods = []string{
		"eth_getTransactionReceipt",
	}
)

func CallType(t string) bool {
	callTypes := []string{
		OpCall,
		OpCallCode,
		OpDelegateCall,
		OpStaticCall,
	}

	for _, callType := range callTypes {
		if callType == t {
			return true
		}
	}

	return false
}

func CreateType(t string) bool {
	createTypes := []string{
		OpCreate,
		OpCreate2,
	}

	for _, createType := range createTypes {
		if createType == t {
			return true
		}
	}

	return false
}

func ToCurrency(symbol string, decimals uint8, contractAddress common.Address) *types.Currency {
	return &types.Currency{
		Symbol:   symbol,
		Decimals: int32(decimals),
		Metadata: map[string]interface{}{
			ContractAddressMetadata: contractAddress.Hex(),
		},
	}
}
