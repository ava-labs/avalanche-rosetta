package service

import (
	"os"
	"testing"

	"github.com/ava-labs/coreth/plugin/evm"
)

func TestMain(m *testing.M) {
	evm.RegisterAllLibEVMExtras()
	os.Exit(m.Run())
}
