package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/ava-labs/avalanchego/utils/cb58"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/coinbase/rosetta-sdk-go/types"
)

type signRequest struct {
	Payloads []*types.SigningPayload `json:"payloads"`
}

type signResponse struct {
	Signatures []*types.Signature `json:"signatures"`
}

type signingServer struct {
	port       uint
	privateKey *secp256k1.PrivateKey
	publicKey  *secp256k1.PublicKey
}

func (s *signingServer) SignBytes(address string, bytes []byte) (*types.Signature, error) {
	signatureBytes, err := s.privateKey.SignHash(bytes)
	if err != nil {
		return nil, err
	}

	return &types.Signature{
		SigningPayload: &types.SigningPayload{
			AccountIdentifier: &types.AccountIdentifier{Address: address},
			Bytes:             bytes,
			SignatureType:     types.EcdsaRecovery,
		},
		PublicKey: &types.PublicKey{
			Bytes:     s.publicKey.Bytes(),
			CurveType: types.Secp256k1,
		},
		SignatureType: types.EcdsaRecovery,
		Bytes:         signatureBytes,
	}, nil
}

func (s *signingServer) signHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		d := json.NewDecoder(r.Body)
		signPayload := signRequest{}
		err := d.Decode(&signPayload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		log.Printf("Received signing request with %d payloads\n", len(signPayload.Payloads))

		var signatures []*types.Signature

		for _, payload := range signPayload.Payloads {
			signature, err := s.SignBytes(payload.AccountIdentifier.Address, payload.Bytes)
			if err != nil {
				http.Error(w, "unable to sign payload", 500)
				return
			}

			signatures = append(signatures, signature)
		}

		resp := signResponse{Signatures: signatures}
		bytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "unable to marshal response", 500)
			return
		}

		_, _ = w.Write(bytes)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *signingServer) run() {
	http.HandleFunc("/sign", s.signHandler)

	addr := fmt.Sprintf("localhost:%d", s.port)
	log.Printf("Listening on %s\n", addr)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func parsePrivateKeyString(keyStr string) (*secp256k1.PrivateKey, *secp256k1.PublicKey, error) {
	parts := strings.Split(keyStr, "-")
	pkBytes, err := cb58.Decode(parts[1])
	if err != nil {
		return nil, nil, err
	}
	factory := secp256k1.Factory{}
	key, err := factory.ToPrivateKey(pkBytes)
	if err != nil {
		return nil, nil, err
	}

	pk := key.PublicKey()
	return key, pk, nil
}

func newSigningServer(port uint, privateKeyStr string) *signingServer {
	privateKey, publicKey, err := parsePrivateKeyString(privateKeyStr)
	if err != nil {
		log.Fatal(err)
	}

	return &signingServer{
		privateKey: privateKey,
		publicKey:  publicKey,
		port:       port,
	}
}

var opts struct {
	port       uint
	privateKey string
}

func init() {
	flag.UintVar(&opts.port, "port", 9898, "signing server port")
	flag.StringVar(&opts.privateKey, "private-key", "", "private key to sign with")
	flag.Parse()
}

func main() {
	println("!!! DO NOT USE THIS UTILITY WITH PRIVATE KEYS THAT HAS ACCESS TO REAL FUNDS!!!\n")
	println("This is a test utility used to help testing Rosetta AVAX implementation during development\n\n")
	signingServer := newSigningServer(opts.port, opts.privateKey)
	signingServer.run()
}
