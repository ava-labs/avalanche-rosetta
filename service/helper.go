package service

import (
	"strings"

	"github.com/coinbase/rosetta-sdk-go/types"

	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	nativeTransferGasLimit = uint64(21000)
	erc20TransferGasLimit  = uint64(250000)
	genesisTimestamp       = 946713601000 // min allowable timestamp
)

func makeGenesisBlock(hash string) *types.Block {
	return &types.Block{
		BlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		ParentBlockIdentifier: &types.BlockIdentifier{
			Index: 0,
			Hash:  hash,
		},
		Timestamp: genesisTimestamp,
	}
}

// ChecksumAddress ensures an Ethereum hex address
// is in Checksum Format. If the address cannot be converted,
// it returns !ok.
func ChecksumAddress(address string) (string, bool) {
	if !strings.HasPrefix(address, "0x") {
		return "", false
	}

	addr, err := ethcommon.NewMixedcaseAddressFromString(address)
	if err != nil {
		return "", false
	}

	return addr.Address().Hex(), true
}
