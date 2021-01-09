package mapper

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/coreth/plugin/evm"
)

var (
	preApricotCodecVersion uint16 = 0
	apricotCodecVersion    uint16 = 1

	codecManager codec.Manager
)

func init() {
	codecManager = codec.NewDefaultManager()

	preApricotCodec := linearcodec.NewDefault()
	preApricotCodec.RegisterType(&evm.UnsignedImportTx{})
	preApricotCodec.RegisterType(&evm.UnsignedExportTx{})
	preApricotCodec.SkipRegistrations(3)
	preApricotCodec.RegisterType(&secp256k1fx.TransferInput{})
	preApricotCodec.RegisterType(&secp256k1fx.MintOutput{})
	preApricotCodec.RegisterType(&secp256k1fx.TransferOutput{})
	preApricotCodec.RegisterType(&secp256k1fx.MintOperation{})
	preApricotCodec.RegisterType(&secp256k1fx.Credential{})
	preApricotCodec.RegisterType(&secp256k1fx.Input{})
	preApricotCodec.RegisterType(&secp256k1fx.OutputOwners{})

	codecManager.RegisterCodec(preApricotCodecVersion, preApricotCodec)
}
