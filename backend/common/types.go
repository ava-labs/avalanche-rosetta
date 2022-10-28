package common

import (
	"encoding/json"
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

var ErrNoTxGiven = errors.New("no transaction was given")

// AvaxTx encapsulates P-chain and C-chain atomic transactions in order to reuse common logic between them
type AvaxTx interface {
	Initialize() error
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	SigningPayload() []byte
	Hash() ids.ID
}

// RosettaTx wraps a transaction along with the input addresses and destination chain information.
// It is used during construction between /construction/payloads and /construction/parse since parse needs this information
// but C-chain atomic and P-chain tx formats strip this information and only retain UTXO ids.
type RosettaTx struct {
	Tx                       AvaxTx
	AccountIdentifierSigners []Signer
	DestinationChain         string
	DestinationChainID       *ids.ID
}

// Signer contains details of coin identifiers and the accounts signing those coins
type Signer struct {
	CoinIdentifier    string                   `json:"coin_identifier,omitempty"`
	AccountIdentifier *types.AccountIdentifier `json:"account_identifier"`
}

type rosettaTxWire struct {
	Tx                 string   `json:"tx"`
	Signers            []Signer `json:"signers"`
	DestinationChain   string   `json:"destination_chain,omitempty"`
	DestinationChainID *ids.ID  `json:"destination_chain_id,omitempty"`
}

func (t *RosettaTx) MarshalJSON() ([]byte, error) {
	bytes, err := t.Tx.Marshal()
	if err != nil {
		return nil, err
	}

	str, err := mapper.EncodeBytes(bytes)
	if err != nil {
		return nil, err
	}

	txWire := &rosettaTxWire{
		Tx:                 str,
		Signers:            t.AccountIdentifierSigners,
		DestinationChain:   t.DestinationChain,
		DestinationChainID: t.DestinationChainID,
	}
	return json.Marshal(txWire)
}

func (t *RosettaTx) UnmarshalJSON(data []byte) error {
	if t.Tx == nil {
		return errors.New("tx must be initialized before unmarshalling")
	}
	txWire := &rosettaTxWire{}
	err := json.Unmarshal(data, txWire)
	if err != nil {
		return err
	}

	bytes, err := mapper.DecodeToBytes(txWire.Tx)
	if err != nil {
		return err
	}

	err = t.Tx.Unmarshal(bytes)
	if err != nil {
		return err
	}

	t.AccountIdentifierSigners = txWire.Signers
	t.DestinationChain = txWire.DestinationChain
	t.DestinationChainID = txWire.DestinationChainID

	return nil
}

// GetAccountIdentifiers extracts input account identifiers from given Rosetta operations
func (t *RosettaTx) GetAccountIdentifiers(operations []*types.Operation) ([]*types.AccountIdentifier, error) {
	signers := []*types.AccountIdentifier{}

	operationToAccountMap := make(map[string]*types.AccountIdentifier)
	for _, data := range t.AccountIdentifierSigners {
		operationToAccountMap[data.CoinIdentifier] = data.AccountIdentifier
	}

	for _, op := range operations {
		// Skip positive amounts
		if op.Amount.Value[0] != '-' {
			continue
		}

		var coinIdentifier string
		if op.CoinChange != nil && op.CoinChange.CoinIdentifier != nil {
			coinIdentifier = op.CoinChange.CoinIdentifier.Identifier
		}

		signer := operationToAccountMap[coinIdentifier]
		if signer == nil {
			return nil, errors.New("not all operations have signers")
		}
		signers = append(signers, signer)
	}

	return signers, nil
}
