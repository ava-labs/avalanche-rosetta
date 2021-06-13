package mapper

import (
	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/codec/linearcodec"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/coreth/plugin/evm"
)

const (
	preApricotCodecVersion uint16 = 0
	codecRegistrationSkip  int    = 3
)

var (
	codecManager codec.Manager
)

func init() {
	codecManager = codec.NewDefaultManager()

	errs := wrappers.Errs{}
	preApricotCodec := initPreApricotCodec(&errs)
	errs.Add(
		codecManager.RegisterCodec(preApricotCodecVersion, preApricotCodec),
	)

	if errs.Errored() {
		panic(errs.Err)
	}
}

func initPreApricotCodec(errs *wrappers.Errs) linearcodec.Codec {
	c := linearcodec.NewDefault()

	errs.Add(
		c.RegisterType(&evm.UnsignedImportTx{}),
		c.RegisterType(&evm.UnsignedExportTx{}),
	)

	c.SkipRegistrations(codecRegistrationSkip)

	errs.Add(
		c.RegisterType(&secp256k1fx.TransferInput{}),
		c.RegisterType(&secp256k1fx.MintOutput{}),
		c.RegisterType(&secp256k1fx.TransferOutput{}),
		c.RegisterType(&secp256k1fx.MintOperation{}),
		c.RegisterType(&secp256k1fx.Credential{}),
		c.RegisterType(&secp256k1fx.Input{}),
		c.RegisterType(&secp256k1fx.OutputOwners{}),
	)

	return c
}
