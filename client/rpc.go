package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/rpc/v2/json2"
)

// RPC is a generic client
type RPC struct {
	endpoint string
	client   *http.Client
}

// Dial returns a new RPC client
func Dial(endpoint string) *RPC {
	return &RPC{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: time.Second * 10,
		},
	}
}

// CallRaw performs the call and returns the raw response data
func (c RPC) CallRaw(ctx context.Context, method string, args interface{}) ([]byte, error) {
	data, err := json2.EncodeClientRequest(method, args)
	if err != nil {
		return nil, err
	}
	reqBody := bytes.NewReader(data)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

// Call performs the call and decodes the response into the target interface
func (c RPC) Call(ctx context.Context, method string, args interface{}, out interface{}) error {
	data, err := c.CallRaw(ctx, method, args)
	if err != nil {
		return err
	}
	return json2.DecodeClientResponse(bytes.NewReader(data), out)
}
