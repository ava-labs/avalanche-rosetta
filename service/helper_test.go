package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChecksumAddress(t *testing.T) {
	t.Run("valid checksum address", func(t *testing.T) {
		testAddr := "0x05da63494DfbfF6AA215E074D34aC9A25B616eF2"
		addr, ok := ChecksumAddress(testAddr)
		require.True(t, ok)
		require.Equal(t, testAddr, addr)
	})

	t.Run("modified checksum address", func(t *testing.T) {
		testAddr := "0x05da63494DfbfF6AA215E074D34aC9A25B616ef2"
		addr, ok := ChecksumAddress(testAddr)
		require.True(t, ok)
		require.Equal(t, "0x05da63494DfbfF6AA215E074D34aC9A25B616eF2", addr)
	})

	t.Run("invalid hex", func(t *testing.T) {
		testAddr := "0x05da63494DfbfF6AA215E074D34aC9A25B616eK2"
		addr, ok := ChecksumAddress(testAddr)
		require.False(t, ok)
		require.Equal(t, "", addr)
	})

	t.Run("invalid length", func(t *testing.T) {
		testAddr := "0x05da63494DfbfF6AA215E074D34aC9A25B"
		addr, ok := ChecksumAddress(testAddr)
		require.False(t, ok)
		require.Equal(t, "", addr)
	})

	t.Run("missing 0x", func(t *testing.T) {
		testAddr := "05da63494DfbfF6AA215E074D34aC9A25B616eF2"
		addr, ok := ChecksumAddress(testAddr)
		require.False(t, ok)
		require.Equal(t, "", addr)
	})
}
