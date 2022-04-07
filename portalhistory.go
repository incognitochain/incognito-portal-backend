package main

import (
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"log"
	"sync"
)

// PortalShieldHistory represents the history of a shielding request.
type PortalShieldHistory struct {
	// Amount is the shielding amount.
	Amount uint64 `json:"amount,omitempty"`

	// ExternalTxID is the hash of the public transaction sending the corresponding public token to the multi-sig address.
	ExternalTxID string `json:"externalTxID"`

	// IncognitoAddress is the payment address associated with the request (if have).
	IncognitoAddress string `json:"incognitoAddress,omitempty"`

	// DepositPubKey is the OTDepositPubKey associated with the request (if have).
	DepositPubKey string `json:"depositpubkey,omitempty"`

	// Status can be one of the following:
	//	- 0: FAILED
	//	- 1: SUCCEEDED
	//	- 2: PENDING
	//	- 3: PROCESSING
	Status int `json:"status"`

	// Time is the timestamp of the shielding request.
	Time int64 `json:"time,omitempty"`

	// Confirmations is the number of confirmations of `ExternalTxID`.
	Confirmations int64 `json:"confirmations"`
}

const ShieldStatusFailed = 0
const ShieldStatusSuccess = 1
const ShieldStatusPending = 2
const ShieldStatusProcessing = 3

func convertBTCAmtToPBTCAmt(btcAmt float64) uint64 {
	return uint64(btcAmt*1e8+0.5) * 10
}

func getStatusFromConfirmation(confirmationBlks int) (status int) {
	status = ShieldStatusPending
	if confirmationBlks > 0 {
		status = ShieldStatusProcessing
	}
	return
}

func getShieldHistoryForKey(depositKey string, useOTDepositKey ...bool) ([]PortalShieldHistory, error) {
	btcAddressStr, err := DBGetBTCAddressByIncAddress(depositKey, useOTDepositKey...)
	if err != nil {
		return nil, fmt.Errorf("could not get btc address by chainCode %v from DB", depositKey)
	}

	btcAddress, err := btcutil.DecodeAddress(btcAddressStr, BTCChainCfg)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not decode address %v - with err: %v", btcAddressStr, err))
		return nil, fmt.Errorf("could not decode address %v - with err: %v", btcAddressStr, err)
	}

	// time1 := time.Now()
	utxos, err := btcClient.ListUnspentMinMaxAddresses(BTCMinConf, BTCMaxConf, []btcutil.Address{btcAddress})
	if err != nil {
		log.Printf(fmt.Sprintf("could not get utxos of address %v - with err: %v", btcAddressStr, err))
		return nil, fmt.Errorf("could not get utxos of address %v - with err: %v", btcAddressStr, err)
	}

	// time2 := time.Now()
	history, err := ParseUTXOsToPortalShieldHistory(utxos, depositKey, useOTDepositKey...)
	if err != nil {
		log.Printf(fmt.Sprintf("could not get history from utxos of address %v - with err: %v", btcAddressStr, err))
		return nil, fmt.Errorf("could not get history from utxos of address  %v - with err: %v", btcAddressStr, err)
	}

	return history, nil
}

func ParseUTXOsToPortalShieldHistory(
	utxos []btcjson.ListUnspentResult, depositKey string, useOTDepositKey ...bool,
) ([]PortalShieldHistory, error) {
	histories := make([]PortalShieldHistory, 0)

	var wg sync.WaitGroup
	result := make(chan PortalShieldHistory, len(utxos))
	for _, u := range utxos {
		u := u
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := getStatusFromConfirmation(int(u.Confirmations))
			txIDHash, err := chainhash.NewHashFromStr(u.TxID)
			if err != nil {
				log.Printf("Could not new hash from external tx id %v - Error %v\n", u.TxID, err)
				return
			}
			tx, err := btcClient.GetTransaction(txIDHash)
			if err != nil {
				log.Printf("Could not get external tx id %v - Error %v\n", u.TxID, err)
				return
			}

			h := PortalShieldHistory{
				Amount:        convertBTCAmtToPBTCAmt(u.Amount),
				ExternalTxID:  u.TxID,
				Status:        status,
				Time:          tx.Time * 1000, // convert to msec
				Confirmations: u.Confirmations,
			}
			if len(useOTDepositKey) > 0 && useOTDepositKey[0] {
				h.DepositPubKey = depositKey
			} else {
				h.IncognitoAddress = depositKey
			}
			result <- h
		}()
	}
	wg.Wait()
	close(result)

	for h := range result {
		histories = append(histories, h)
	}

	return histories, nil
}
