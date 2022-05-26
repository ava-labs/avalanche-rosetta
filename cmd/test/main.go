package main

// import (
// 	"context"
// 	"crypto/tls"
// 	"fmt"
// 	"net"
// 	"net/http"
// 	"time"

// 	rosettaclient "github.cbhq.net/c3/keychain/pkg/rosetta"
// 	"github.com/coinbase/rosetta-sdk-go/types"

// 	"github.com/ava-labs/avalanchego/utils/crypto"
// )

// func main() {
// 	fmt.Println("test")
// 	rosettaClient := newRosettaClient()
// 	factory := crypto.FactorySECP256K1R{}
// 	priv, nil := factory.NewPrivateKey()
// 	res, err := rosettaClient.DeriveFromPublicKey(context.Background(), &types.ConstructionDeriveRequest{
// 		NetworkIdentifier: &types.NetworkIdentifier{
// 			Blockchain: "Avalanche",
// 			Network:    "Fuji",
// 			SubNetworkIdentifier: &types.SubNetworkIdentifier{
// 				Network: "P",
// 			},
// 		},
// 		PublicKey: &types.PublicKey{
// 			Bytes:     priv.PublicKey().Bytes(),
// 			CurveType: types.Secp256r1,
// 		},
// 	})
// 	if err != nil {
// 		panic(err)
// 	}
// 	fmt.Println(res)
// }

// func newRosettaClient() *rosettaclient.Client {
// 	var (
// 		dialer = &net.Dialer{Timeout: 10}
// 		// #nosec G402
// 		tlsConfig  = &tls.Config{InsecureSkipVerify: true}
// 		url        = "https://localhost:8080"
// 		httpClient = &http.Client{
// 			Timeout: 10 * time.Second,
// 			Transport: &http.Transport{
// 				Dial:                dialer.Dial,
// 				TLSHandshakeTimeout: 10 * time.Second,
// 				TLSClientConfig:     tlsConfig,
// 			},
// 		}
// 	)

// 	return rosettaclient.NewClient(httpClient, url, false)
// }

// // curl --data '{"network_identifier":{"blockchain":"Avalanche","network":"Fuji", "sub_network_identifier":{"network":"P"}},"public_key":{"hex_bytes":"02d95fcfa4321a1291bd8b387635fe0c3b9fa6d243153387fbe298a467b0c2ab32","curve_type":"secp256r1"}}' --header "Content-Type: application/json" --request POST 0.0.0.0:8080/construction/derive | jq .
