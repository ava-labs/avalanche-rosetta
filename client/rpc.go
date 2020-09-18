package client

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/rpc/v2/json2"
	"github.com/sirupsen/logrus"
)

const (
	PrefixInfo     = "/ext/info"
	PrefixPlatform = "/ext/P"
	PrefixAVM      = "/ext/bc/X"
	PrefixEVM      = "/ext/bc/C/rpc"
)

// RPC implements a generic JSON-RPCv2 client
type RPC struct {
	endpoint string
	client   *http.Client
	logger   *logrus.Logger
}

func NewRPCClient(endpoint string) RPC {
	return RPC{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

func (c RPC) CallRaw(method string, args interface{}) ([]byte, error) {
	data, err := json2.EncodeClientRequest(method, args)
	if err != nil {
		return nil, err
	}
	reqBody := bytes.NewReader(data)

	req, err := http.NewRequest(http.MethodPost, c.endpoint, reqBody)
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

func (c RPC) Call(method string, args interface{}, out interface{}) error {
	data, err := c.CallRaw(method, args)
	if err != nil {
		return err
	}
	return json2.DecodeClientResponse(bytes.NewReader(data), out)
}
