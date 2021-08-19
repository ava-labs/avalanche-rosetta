package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChecksumAddress(t *testing.T) {
	t.Run("valid checksum address", func(t *testing.T) {
		testAddr := "0x05da63494DfbfF6AA215E074D34aC9A25B616eF2"
		addr, ok := ChecksumAddress(testAddr)
		assert.True(t, ok)
		assert.Equal(t, testAddr, addr)
	})

	t.Run("modified checksum address", func(t *testing.T) {
		testAddr := "0x05da63494DfbfF6AA215E074D34aC9A25B616ef2"
		addr, ok := ChecksumAddress(testAddr)
		assert.True(t, ok)
		assert.Equal(t, "0x05da63494DfbfF6AA215E074D34aC9A25B616eF2", addr)
	})

	t.Run("invalid hex", func(t *testing.T) {
		testAddr := "0x05da63494DfbfF6AA215E074D34aC9A25B616eK2"
		addr, ok := ChecksumAddress(testAddr)
		assert.False(t, ok)
		assert.Equal(t, "", addr)
	})
}
