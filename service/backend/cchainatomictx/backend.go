package cchainatomictx

import (
	"github.com/ava-labs/avalanche-rosetta/service"
)

var (
	_ service.ConstructionBackend = &Backend{}
	_ service.AccountBackend      = &Backend{}
)

type Backend struct{}

func NewBackend() (*Backend, error) {
	return &Backend{}, nil
}

func (b *Backend) ShouldHandleRequest(req interface{}) bool {
	return false
}
