package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/constants"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
	"github.com/ava-labs/avalanche-rosetta/service/backend/cchainatomictx"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain"
	"github.com/ava-labs/avalanche-rosetta/service/backend/pchain/indexer"

	pmapper "github.com/ava-labs/avalanche-rosetta/mapper/pchain"
)

var (
	cmdName    = "avalanche-rosetta"
	cmdVersion = service.MiddlewareVersion
)

var opts struct {
	configPath string
	version    bool
}

func init() {
	flag.StringVar(&opts.configPath, "config", "", "Path to configuration file")
	flag.BoolVar(&opts.version, "version", false, "Print version")
	flag.Parse()
}

func main() {
	if opts.version {
		log.Printf("%s %s\n", cmdName, cmdVersion)
		return
	}

	if opts.configPath == "" {
		log.Fatal("config file is not provided")
	}

	cfg, err := readConfig(opts.configPath)
	if err != nil {
		log.Fatal("config read error:", err)
	}

	// set defaults for unspecified configs
	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		log.Fatal("config validation error:", err)
	}

	cChainClient, err := client.NewClient(context.Background(), cfg.RPCBaseURL)
	if err != nil {
		log.Fatal("client init error:", err)
	}

	// [ValidateERC20Whitelist] is disabled by default because it requires
	// a fully synced node to work correctly. If the underlying node is still
	// bootstrapping, it will fail.
	//
	// TODO: Only perform this check after the underlying node is bootstrapped
	if cfg.Mode == service.ModeOnline && cfg.ValidateERC20Whitelist {
		if err := cfg.validateWhitelistOnlyValidErc20s(cChainClient); err != nil {
			log.Fatal("token whitelist validation error:", err)
		}
	}

	log.Println("starting server in", cfg.Mode, "mode")

	if cfg.ChainID == 0 {
		log.Println("chain id is not provided, fetching from rpc...")
		chainID, err := cChainClient.ChainID(context.Background())
		if err != nil {
			log.Fatal("cant fetch chain id from rpc:", err)
		}
		cfg.ChainID = chainID.Int64()
	}

	var assetID string
	var AP5Activation uint64
	switch cfg.ChainID {
	case constants.MainnetChainID:
		assetID = constants.MainnetAssetID
		AP5Activation = constants.MainnetAP5Activation
	case constants.FujiChainID:
		assetID = constants.FujiAssetID
		AP5Activation = constants.FujiAP5Activation
	default:
		log.Fatal("invalid ChainID:", cfg.ChainID)
	}

	// Note: Rosetta is currently configure with capitalized NetworkNames
	// and service network requests are carried our with capital case.
	// while avalanchego requires lower-case network names.
	// We convert to lower case upon specific calls to avalanchego clients.
	networkP := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    cfg.NetworkName,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: constants.PChain.String(),
		},
	}
	networkC := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    cfg.NetworkName,
	}

	avaxAssetID, err := ids.FromString(assetID)
	if err != nil {
		log.Fatal("parse asset id failed:", err)
	}

	pChainClient := client.NewPChainClient(context.Background(), cfg.RPCBaseURL, cfg.IndexerBaseURL)
	pIndexerParser, err := indexer.NewParser(pChainClient, cfg.avalancheNetworkID())
	if err != nil {
		log.Fatal("unable to initialize p-chain indexer parser:", err)
	}

	pChainBackend, err := pchain.NewBackend(
		cfg.Mode,
		pChainClient,
		pIndexerParser,
		avaxAssetID,
		networkP,
		cfg.avalancheNetworkID(),
	)
	if err != nil {
		log.Fatal("unable to initialize p-chain backend:", err)
	}

	cChainAtomicTxBackend := cchainatomictx.NewBackend(cChainClient, avaxAssetID, cfg.avalancheNetworkID())

	serviceConfig := &service.Config{
		Mode:               cfg.Mode,
		ChainID:            big.NewInt(cfg.ChainID),
		NetworkID:          networkC,
		GenesisBlockHash:   cfg.GenesisBlockHash,
		AvaxAssetID:        assetID,
		AP5Activation:      AP5Activation,
		IndexUnknownTokens: cfg.IndexUnknownTokens,
		IngestionMode:      cfg.IngestionMode,
		TokenWhiteList:     cfg.TokenWhiteList,
		BridgeTokenList:    cfg.BridgeTokenList,
	}

	var operationTypes []string
	operationTypes = append(operationTypes, mapper.OperationTypes...)
	operationTypes = append(operationTypes, pmapper.OperationTypes...)

	asserter, err := asserter.NewServer(
		operationTypes, // supported operation types
		true,           // historical balance lookup
		[]*types.NetworkIdentifier{ // supported networks
			networkP,
			networkC,
		}, // supported networks
		[]string{}, // call methods
		false,      // mempool coins
	)
	if err != nil {
		log.Fatal("server asserter init error:", err)
	}

	handler := configureRouter(serviceConfig, asserter, cChainClient, pChainBackend, cChainAtomicTxBackend)
	if cfg.LogRequests {
		handler = inspectMiddleware(handler)
	}
	handler = server.LoggerMiddleware(handler)

	router := server.CorsMiddleware(handler)

	log.Printf(
		`using avax (chain=%q chainid="%d" network=%q) rpc endpoint: %v`,
		service.BlockchainName,
		cfg.ChainID,
		cfg.NetworkName,
		cfg.RPCBaseURL,
	)
	log.Printf("starting rosetta server at %s\n", cfg.ListenAddr)

	log.Fatal(http.ListenAndServe(cfg.ListenAddr, router))
}

func configureRouter(
	serviceConfig *service.Config,
	asserter *asserter.Asserter,
	apiClient client.Client,
	pChainBackend *pchain.Backend,
	cChainAtomicTxBackend *cchainatomictx.Backend,
) http.Handler {
	networkService := service.NewNetworkService(serviceConfig, apiClient, pChainBackend)
	blockService := service.NewBlockService(serviceConfig, apiClient, pChainBackend)
	accountService := service.NewAccountService(serviceConfig, apiClient, pChainBackend, cChainAtomicTxBackend)
	mempoolService := service.NewMempoolService(serviceConfig, apiClient)
	constructionService := service.NewConstructionService(serviceConfig, apiClient, pChainBackend, cChainAtomicTxBackend)
	callService := service.NewCallService(serviceConfig, apiClient)

	return server.NewRouter(
		server.NewNetworkAPIController(networkService, asserter),
		server.NewBlockAPIController(blockService, asserter),
		server.NewAccountAPIController(accountService, asserter),
		server.NewMempoolAPIController(mempoolService, asserter),
		server.NewConstructionAPIController(constructionService, asserter),
		server.NewCallAPIController(callService, asserter),
	)
}

// Inspect middlware used to inspect the body of requets
func inspectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}
		body = bytes.TrimSpace(body)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		log.Printf("[DEBUG] %s %s: %s\n", r.Method, r.URL.Path, body)
		next.ServeHTTP(w, r)
	})
}
