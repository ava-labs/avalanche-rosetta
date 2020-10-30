package service

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/figment-networks/avalanche-rosetta/client"
)

const (
	transferGasLimit = uint64(21000)
)

type unsignedTx struct {
	Nonce    uint64   `json:"nonce"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Amount   *big.Int `json:"value"`
	GasPrice *big.Int `json:"gas_price"`
	GasLimit uint64   `json:"gas"`
	Input    []byte   `json:"input"`
}

type txMetadata struct {
	Nonce    uint64   `json:"nonce"`
	GasPrice *big.Int `json:"gas_price"`
}

func blockHeaderFromInput(evm *client.EvmClient, input *types.PartialBlockIdentifier) (*ethtypes.Header, *types.Error) {
	var (
		header *ethtypes.Header
		err    error
	)

	if input == nil {
		header, err = evm.HeaderByNumber(context.Background(), nil)
	} else {
		if input.Hash == nil && input.Index == nil {
			return nil, errInvalidInput
		}

		if input.Index != nil {
			header, err = evm.HeaderByNumber(context.Background(), big.NewInt(*input.Index))
		} else {
			header, err = evm.HeaderByHash(context.Background(), ethcommon.HexToHash(*input.Hash))
		}
	}

	if err != nil {
		return nil, errorWithInfo(errInternalError, err)
	}

	return header, nil
}

func txFromInput(input string) (*ethtypes.Transaction, error) {
	tx := &ethtypes.Transaction{}
	if err := tx.UnmarshalJSON([]byte(input)); err != nil {
		return nil, err
	}
	return tx, nil
}

func unsignedTxFromInput(input string) (*ethtypes.Transaction, error) {
	tx := unsignedTx{}
	inputBytes := []byte(input)

	if err := json.Unmarshal(inputBytes, &tx); err != nil {
		return nil, err
	}

	ethTx := ethtypes.NewTransaction(
		tx.Nonce,
		ethcommon.HexToAddress(tx.To),
		tx.Amount,
		tx.GasLimit,
		tx.GasPrice,
		inputBytes,
	)

	return ethTx, nil
}

func txFromMatches(matches []*parser.Match, kv map[string]interface{}) (*ethtypes.Transaction, *unsignedTx, error) {
	var metadata txMetadata
	if err := unmarshalJSONMap(kv, &metadata); err != nil {
		return nil, nil, err
	}

	fromOp, _ := matches[0].First()
	fromAddress := fromOp.Account.Address
	toOp, amount := matches[1].First()
	toAddress := toOp.Account.Address
	nonce := metadata.Nonce
	gasPrice := metadata.GasPrice
	data := []byte{}

	tx := ethtypes.NewTransaction(
		metadata.Nonce,
		common.HexToAddress(toAddress),
		amount,
		transferGasLimit,
		gasPrice,
		data,
	)

	unTx := &unsignedTx{
		From:     fromAddress,
		To:       toAddress,
		Nonce:    nonce,
		Amount:   amount,
		GasPrice: gasPrice,
		GasLimit: transferGasLimit,
		Input:    data,
	}

	return tx, unTx, nil
}

func unmarshalJSONMap(m map[string]interface{}, i interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, i)
}
