package common

import (
	"encoding/json"
	"errors"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/mapper"
)

type AvaxTx interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
	SigningPayload() ([]byte, error)
	Hash() (ids.ID, error)
}

type RosettaTx struct {
	// The body of this transaction
	Tx AvaxTx

	// AccountIdentifierSigners used by /construction/parse

	AccountIdentifierSigners []Signer

	DestinationChain   string
	DestinationChainID *ids.ID
}

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
