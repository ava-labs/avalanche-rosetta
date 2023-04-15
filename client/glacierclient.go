package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
)

type GlacierClient interface {
	TransactionsListStaking(ctx context.Context, addresses string) ([]PChainTransaction, error)
}

type GlacierHTTPClient interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

type GlacierClientImpl struct {
	HTTPClient      GlacierHTTPClient
	GlacierEndpoint string
	Network         string
}

func NewGlacierClient(glacierEndpoint string, network string) GlacierClient {
	return &GlacierClientImpl{
		GlacierEndpoint: glacierEndpoint,
		Network:         strings.ToLower(network),
		HTTPClient:      retryableHTTPClient{retryablehttp.NewClient()},
	}
}

type PChainEmittedUtxo struct {
	Addresses []string `json:"addresses,omitempty"`
	Amount    string   `json:"amount,omitempty"`
	AssetID   string   `json:"assetId,omitempty"`
	Staked    bool     `json:"staked,omitempty"`
}

type PChainTransaction struct {
	BlockNumber  string              `json:"blockNumber,omitempty"`
	BlockHash    string              `json:"blockHash,omitempty"`
	TxType       string              `json:"txType,omitempty"`
	TxHash       string              `json:"txHash,omitempty"`
	EmittedUTXOs []PChainEmittedUtxo `json:"emittedUTXOs,omitempty"`
}

type TransactionListStakingResponse struct {
	NextPageToken string              `json:"nextPageToken,omitempty"`
	Transactions  []PChainTransaction `json:"transactions,omitempty"`
}

func (g *GlacierClientImpl) TransactionsListStaking(ctx context.Context, addresses string) ([]PChainTransaction, error) {
	endpoint := fmt.Sprintf("v1/networks/%s/blockchains/p-chain/transactions:listStaking?pageSize=100&sortOrder=asc",
		g.Network)

	transactions := make([]PChainTransaction, 0)

	addresses = url.QueryEscape(addresses)
	pageToken := ""
	for {
		endpointWithParameters := fmt.Sprintf("%s&addresses=%s&pageToken=%s", endpoint, addresses, pageToken)
		response := TransactionListStakingResponse{}
		err := g.makeRequest(ctx, endpointWithParameters, &response)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, response.Transactions...)

		if response.NextPageToken == "" {
			break
		}
		pageToken = url.QueryEscape(response.NextPageToken)
	}

	return transactions, nil
}

func (g *GlacierClientImpl) makeRequest(ctx context.Context, endpoint string, responseObject interface{}) error {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(g.GlacierEndpoint, "/"), endpoint)

	body, err := g.HTTPClient.Get(ctx, url)
	if err != nil {
		return err
	}

	return json.Unmarshal(body, responseObject)
}

type retryableHTTPClient struct {
	httpClient *retryablehttp.Client
}

func (r retryableHTTPClient) Get(ctx context.Context, url string) ([]byte, error) {
	req, err := retryablehttp.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = resp.Body.Close()
	}()

	return io.ReadAll(resp.Body)
}
