package mapper

import (
	"github.com/ava-labs/avalanchego/utils/codec"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/coreth/plugin/evm"
)

var (
	codecManager codec.Manager
)

func init() {
	defCodec := codec.NewDefault()
	defCodec.RegisterType(&evm.UnsignedImportTx{})
	defCodec.RegisterType(&evm.UnsignedExportTx{})
	defCodec.Skip(3)
	defCodec.RegisterType(&secp256k1fx.TransferInput{})
	defCodec.RegisterType(&secp256k1fx.MintOutput{})
	defCodec.RegisterType(&secp256k1fx.TransferOutput{})
	defCodec.RegisterType(&secp256k1fx.MintOperation{})
	defCodec.RegisterType(&secp256k1fx.Credential{})
	defCodec.RegisterType(&secp256k1fx.Input{})
	defCodec.RegisterType(&secp256k1fx.OutputOwners{})

	codecManager = codec.NewDefaultManager()
	codecManager.RegisterCodec(uint16(0), defCodec)
}
