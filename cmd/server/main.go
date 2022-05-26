package main

import (
	"bytes"
	"context"
	"flag"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"

	"github.com/coinbase/rosetta-sdk-go/asserter"
	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

	"github.com/ava-labs/avalanche-rosetta/client"
	"github.com/ava-labs/avalanche-rosetta/mapper"
	"github.com/ava-labs/avalanche-rosetta/service"
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
	if err := cfg.Validate(); err != nil {
		log.Fatal("config validation error:", err)
	}

	apiClient, err := client.NewClient(context.Background(), cfg.RPCEndpoint)
	if err != nil {
		log.Fatal("client init error:", err)
	}

	// [ValidateERC20Whitelist] is disabled by default because it requires
	// a fully synced node to work correctly. If the underlying node is still
	// bootstrapping, it will fail.
	//
	// TODO: Only perform this check after the underlying node is bootstrapped
	if cfg.Mode == service.ModeOnline && cfg.ValidateERC20Whitelist {
		if err := cfg.ValidateWhitelistOnlyValidErc20s(apiClient); err != nil {
			log.Fatal("token whitelist validation error:", err)
		}
	}

	log.Println("starting server in", cfg.Mode, "mode")

	if cfg.ChainID == 0 {
		log.Println("chain id is not provided, fetching from rpc...")

		if cfg.Mode == service.ModeOffline {
			log.Fatal("cant fetch chain id in offline mode")
		}

		chainID, err := apiClient.ChainID(context.Background())
		if err != nil {
			log.Fatal("cant fetch chain id from rpc:", err)
		}
		cfg.ChainID = chainID.Int64()
	}

	var assetID string
	var AP5Activation uint64
	switch cfg.ChainID {
	case mapper.MainnetChainID:
		assetID = mapper.MainnetAssetID
		AP5Activation = mapper.MainnetAP5Activation.Uint64()
	case mapper.FujiChainID:
		assetID = mapper.FujiAssetID
		AP5Activation = mapper.FujiAP5Activation.Uint64()
	default:
		log.Fatal("invalid ChainID:", cfg.ChainID)
	}

	if cfg.NetworkName == "" {
		log.Println("network name is not provided, fetching from rpc...")

		if cfg.Mode == service.ModeOffline {
			log.Fatal("cant fetch network name in offline mode")
		}

		networkName, err := apiClient.GetNetworkName(context.Background())
		if err != nil {
			log.Fatal("cant fetch network name:", err)
		}
		cfg.NetworkName = networkName
	}

	networkP := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    cfg.NetworkName,
		SubNetworkIdentifier: &types.SubNetworkIdentifier{
			Network: "P",
		},
	}
	networkC := &types.NetworkIdentifier{
		Blockchain: service.BlockchainName,
		Network:    cfg.NetworkName,
	}

	asserter, err := asserter.NewServer(
		mapper.OperationTypes, // supported operation types
		true,                  // historical balance lookup
		[]*types.NetworkIdentifier{networkP, networkC}, // supported networks
		[]string{}, // call methods
		false,      // mempool coins
	)
	if err != nil {
		log.Fatal("server asserter init error:", err)
	}

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
	}

	handler := configureRouter(serviceConfig, asserter, apiClient)
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
		cfg.RPCEndpoint,
	)
	log.Printf("starting rosetta server at %s\n", cfg.ListenAddr)

	log.Fatal(http.ListenAndServe(cfg.ListenAddr, router))
}

func configureRouter(
	serviceConfig *service.Config,
	asserter *asserter.Asserter,
	apiClient client.Client,
) http.Handler {
	networkService := service.NewNetworkService(serviceConfig, apiClient)
	blockService := service.NewBlockService(serviceConfig, apiClient)
	accountService := service.NewAccountService(serviceConfig, apiClient)
	mempoolService := service.NewMempoolService(serviceConfig, apiClient)
	constructionService := service.NewConstructionService(serviceConfig, apiClient)
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
