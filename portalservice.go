package main

import (
	"fmt"

	"github.com/btcsuite/btcd/rpcclient"
	resty "github.com/go-resty/resty/v2"
)

var btcClient *rpcclient.Client

type BlockchainFeeResponse struct {
	Result float64
	Error  error
}

func initPortalService() {
	err := DBCreatePortalAddressIndex()
	if err != nil {
		panic(err)
	}

	connCfg := &rpcclient.ConnConfig{
		Host:         serviceCfg.BTCFullnode.Address,
		User:         serviceCfg.BTCFullnode.User,
		Pass:         serviceCfg.BTCFullnode.Password,
		HTTPPostMode: true,                          // Bitcoin core only supports HTTP POST mode
		DisableTLS:   !serviceCfg.BTCFullnode.Https, // Bitcoin core does not provide TLS by default
	}
	btcClient, err = rpcclient.New(connCfg, nil)
	if err != nil {
		panic(err)
	}

}

func importBTCAddressToFullNode(btcAddress string) error {
	err := btcClient.ImportAddressRescan(btcAddress, "", false)
	return err
}

func getBitcoinFee() (float64, error) {
	client := resty.New()

	response, err := client.R().
		Get(serviceCfg.BlockchainFeeHost)

	if err != nil {
		return 0, err
	}
	if response.StatusCode() != 200 {
		return 0, fmt.Errorf("Response status code: %v", response.StatusCode())
	}
	var responseBody BlockchainFeeResponse
	err = json.Unmarshal(response.Body(), &responseBody)
	if err != nil {
		return 0, fmt.Errorf("Could not parse response: %v", response.Body())
	}
	return responseBody.Result, nil
}
