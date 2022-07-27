package common

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/platformvm"
)

// initializes tx to have tx identifier generated
func InitializeTx(version uint16, c codec.Manager, tx platformvm.Tx) error {
	errs := wrappers.Errs{}

	unsignedBytes, err := c.Marshal(version, &tx.UnsignedTx)
	errs.Add(err)

	signedBytes, err := c.Marshal(version, &tx)
	errs.Add(err)

	tx.Initialize(unsignedBytes, signedBytes)

	return errs.Err
}
