package main

import (
	"crypto/sha256"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	resty "github.com/go-resty/resty/v2"
	"github.com/incognitochain/go-incognito-sdk-v2/wallet"
)

var btcClient *rpcclient.Client

type BlockchainFeeResponse struct {
	Result float64
	Error  error
}

type NumberReqsResult struct {
	TotalReqs uint64
	MaxReqs   uint64
}

type NumberReqsResponse struct {
	Result *NumberReqsResult
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

func generateOTMultisigAddress(masterPubKeys [][]byte, numSigsRequired int, chainCodeSeed string, chainParam *chaincfg.Params) ([]byte, string, error) {
	if len(masterPubKeys) < numSigsRequired || numSigsRequired < 0 {
		return []byte{}, "", fmt.Errorf("Invalid signature requirement")
	}

	pubKeys := [][]byte{}
	// this Incognito address is marked for the address that received change UTXOs
	if chainCodeSeed == "" {
		pubKeys = masterPubKeys[:]
	} else {
		chainCode := chainhash.HashB([]byte(chainCodeSeed))
		for idx, masterPubKey := range masterPubKeys {
			// generate BTC child public key for this Incognito address
			extendedBTCPublicKey := hdkeychain.NewExtendedKey(chainParam.HDPublicKeyID[:], masterPubKey, chainCode, []byte{}, 0, 0, false)
			extendedBTCChildPubKey, _ := extendedBTCPublicKey.Child(0)
			childPubKey, err := extendedBTCChildPubKey.ECPubKey()
			if err != nil {
				return []byte{}, "", fmt.Errorf("Master BTC Public Key (#%v) %v is invalid - Error %v", idx, masterPubKey, err)
			}
			pubKeys = append(pubKeys, childPubKey.SerializeCompressed())
		}
	}

	// create redeem script for m of n multi-sig
	builder := txscript.NewScriptBuilder()
	// add the minimum number of needed signatures
	builder.AddOp(byte(txscript.OP_1 - 1 + numSigsRequired))
	// add the public key to redeem script
	for _, pubKey := range pubKeys {
		builder.AddData(pubKey)
	}
	// add the total number of public keys in the multi-sig script
	builder.AddOp(byte(txscript.OP_1 - 1 + len(pubKeys)))
	// add the check-multi-sig op-code
	builder.AddOp(txscript.OP_CHECKMULTISIG)

	redeemScript, err := builder.Script()
	if err != nil {
		return []byte{}, "", fmt.Errorf("Could not build script - Error %v", err)
	}

	// generate P2WSH address
	scriptHash := sha256.Sum256(redeemScript)
	addr, err := btcutil.NewAddressWitnessScriptHash(scriptHash[:], chainParam)
	if err != nil {
		return []byte{}, "", fmt.Errorf("Could not generate address from script - Error %v", err)
	}
	addrStr := addr.EncodeAddress()

	return redeemScript, addrStr, nil
}

func generateBTCAddress(incAddress string) (string, error) {
	_, address, err := generateOTMultisigAddress(masterPubKeys, numSigsRequired, incAddress, chainCfg)
	if err != nil {
		return "", err
	}
	return address, nil
}

func isValidPortalAddressPair(incAddress string, btcAddress string) error {
	_, err := wallet.Base58CheckDeserialize(incAddress)
	if err != nil {
		return err
	}

	generatedBTCAddress, err := generateBTCAddress(incAddress)
	if err != nil {
		return err
	}
	if generatedBTCAddress != btcAddress {
		return fmt.Errorf("Invalid BTC address")
	}

	return nil
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

// call backend service to get number of request by api name of an access token
func getNumReqsByAPI(apiName, accessToken string) (*NumberReqsResult, error) {
	client := resty.New()

	req := client.R()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("x-api-token", accessToken)
	req.Header.Add("x-api-key", serviceCfg.BackendKey)

	response, err := req.Get(serviceCfg.BackendServiceHost + "/security/get-reqs-number/" + apiName)
	if err != nil {
		return nil, err
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("Response getNumReqsByAPI status code: %v", response.StatusCode())
	}
	var responseBody NumberReqsResponse
	err = json.Unmarshal(response.Body(), &responseBody)
	if err != nil || responseBody.Result == nil {
		return nil, fmt.Errorf("Could not parse response getNumReqsByAPI: %v", response.Body())
	}

	return responseBody.Result, nil
}

// call backend service to increase number of request by api name of an access token
func increaseNumReqsByAPI(apiName, accessToken string) (*NumberReqsResult, error) {
	client := resty.New()

	req := client.R()
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("x-api-token", accessToken)
	req.Header.Add("x-api-key", serviceCfg.BackendKey)

	response, err := req.Post(serviceCfg.BackendServiceHost + "/security/increase-reqs-number/" + apiName)
	if err != nil {
		return nil, err
	}
	if response.StatusCode() != 200 {
		return nil, fmt.Errorf("Response increaseNumReqsByAPI status code: %v", response.StatusCode())
	}
	var responseBody NumberReqsResponse
	err = json.Unmarshal(response.Body(), &responseBody)
	if err != nil || responseBody.Result == nil {
		return nil, fmt.Errorf("Could not parse response increaseNumReqsByAPI: %v", response.Body())
	}

	return responseBody.Result, nil
}
