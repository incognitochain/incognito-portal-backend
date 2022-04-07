package main

import (
	"flag"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/incognitochain/go-incognito-sdk-v2/incclient"
	"io/ioutil"
)

var ENABLE_PROFILER bool
var serviceCfg Config
var BTCChainCfg *chaincfg.Params
var BTCTokenID string
var incClient *incclient.IncClient

type BTCFullnodeConfig struct {
	Address  string `json:"address"`
	User     string `json:"user"`
	Password string `json:"pass"`
	Https    bool   `json:"https"`
}

type Config struct {
	APIPort           int               `json:"apiport"`
	MongoAddress      string            `json:"mongo"`
	MongoDB           string            `json:"mongodb"`
	BTCFullnode       BTCFullnodeConfig `json:"btcfullnode"`
	BlockchainFeeHost string            `json:"blockchainfee"`
	Net               string            `json:"net"`
}

func readConfigAndArg() {
	data, err := ioutil.ReadFile("./cfg.json")
	if err != nil {
		logger.Println(err)
		// return
	}
	var tempCfg Config
	if data != nil {
		err = json.Unmarshal(data, &tempCfg)
		if err != nil {
			panic(err)
		}
	}

	argProfiler := flag.Bool("profiler", false, "set profiler")
	flag.Parse()
	if tempCfg.APIPort == 0 {
		tempCfg.APIPort = DefaultAPIPort
	}
	if tempCfg.MongoAddress == "" {
		tempCfg.MongoAddress = DefaultMongoAddress
	}
	if tempCfg.MongoDB == "" {
		tempCfg.MongoDB = DefaultMongoDB
	}
	if tempCfg.Net == "test" {
		BTCChainCfg = &chaincfg.TestNet3Params
		BTCTokenID = TESTNET_BTC_ID
		incClient, err = incclient.NewTestNetClient()
		if err != nil {
			panic(err)
		}
	} else if tempCfg.Net == "main" {
		BTCChainCfg = &chaincfg.MainNetParams
		BTCTokenID = MAINNET_BTC_ID
		incClient, err = incclient.NewMainNetClient()
		if err != nil {
			panic(err)
		}
	} else {
		panic("Invalid config network Bitcoin")
	}
	ENABLE_PROFILER = *argProfiler
	serviceCfg = tempCfg
	logger = InitLogger()
}
