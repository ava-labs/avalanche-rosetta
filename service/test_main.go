package service

import (
	"os"
	"testing"

	"github.com/ava-labs/avalanchego/vms/saevm/cchain/libevm"
)

func TestMain(m *testing.M) {
	libevm.RegisterExtras()
	os.Exit(m.Run())
}
