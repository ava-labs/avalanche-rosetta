package client_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanche-rosetta/client"
	mocks "github.com/ava-labs/avalanche-rosetta/mocks/client"
)

func TestTransactionListStaking(t *testing.T) {
	mockHTTPClient := mocks.NewGlacierHTTPClient(t)

	c := client.GlacierClientImpl{
		HTTPClient:      mockHTTPClient,
		GlacierEndpoint: "https://glacierendpoint.avax.network",
		Network:         "mainnet",
	}

	ctx := context.Background()
	address := "P-avax139ckvuc264jkfczumqr5p6p8s97xq2h0qv9hnz"

	url1 := "https://glacierendpoint.avax.network/v1/networks/mainnet/blockchains/p-chain/transactions:listStaking?" +
		"pageSize=100&sortOrder=asc&addresses=P-avax139ckvuc264jkfczumqr5p6p8s97xq2h0qv9hnz&pageToken="
	resp1, err := os.ReadFile("testdata/glacierclient/listStakingResp1.json")
	require.NoError(t, err)

	url2 := "https://glacierendpoint.avax.network/v1/networks/mainnet/blockchains/p-chain/transactions:listStaking?" +
		"pageSize=100&sortOrder=asc&addresses=P-avax139ckvuc264jkfczumqr5p6p8s97xq2h0qv9hnz" +
		"&pageToken=Mzg5OTA0N3x3d0hXOWF4dDREcXVQZ0ZyTnZNdk5ZejlCVlduOGtiemVBR3FXakJHTFJ3Z1BTMWhp%0A"
	resp2, err := os.ReadFile("testdata/glacierclient/listStakingResp2.json")
	require.NoError(t, err)

	mockHTTPClient.On("Get", ctx, url1).Once().Return(resp1, nil)
	mockHTTPClient.On("Get", ctx, url2).Once().Return(resp2, nil)

	transactions, err := c.TransactionsListStaking(ctx, address)
	require.NoError(t, err)
	expectedTransactions := []client.PChainTransaction{
		{
			BlockNumber: "12345",
			BlockHash:   "blockHash1",
			TxType:      "AddValidatorTx",
			TxHash:      "txHash1",
			EmittedUTXOs: []client.PChainEmittedUtxo{
				{
					Addresses: []string{"avax139ckvuc264jkfczumqr5p6p8s97xq2h0qv9hnz"},
					Amount:    "1000000000000",
					AssetID:   "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z",
					Staked:    false,
				},
				{
					Addresses: []string{"avax139ckvuc264jkfczumqr5p6p8s97xq2h0qv9hnz"},
					Amount:    "20000000000000",
					AssetID:   "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z",
					Staked:    true,
				},
			},
		},
		{
			BlockNumber: "24680",
			BlockHash:   "blockHash2",
			TxType:      "AddDelegatorTx",
			TxHash:      "txHash2",
			EmittedUTXOs: []client.PChainEmittedUtxo{
				{
					Addresses: []string{"avax139ckvuc264jkfczumqr5p6p8s97xq2h0qv9hnz"},
					Amount:    "2000000000000",
					AssetID:   "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z",
					Staked:    false,
				},
				{
					Addresses: []string{"avax139ckvuc264jkfczumqr5p6p8s97xq2h0qv9hnz"},
					Amount:    "30000000000000",
					AssetID:   "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z",
					Staked:    true,
				},
			},
		},
	}
	require.Equal(t, expectedTransactions, transactions)
}
