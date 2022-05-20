package main

import (
	"fmt"
	"github.com/incognitochain/go-incognito-sdk-v2/coin"
	"github.com/incognitochain/go-incognito-sdk-v2/common"
	"github.com/incognitochain/go-incognito-sdk-v2/common/base58"
	"github.com/incognitochain/go-incognito-sdk-v2/crypto"
	"github.com/incognitochain/go-incognito-sdk-v2/privacy"
)

// APIAddPortalShieldingRequestParams represents a list of parameters for the `addportalshieldingaddress` api.
// A valid request must satisfy the following conditions:
//	- Either IncAddress or OTDepositPubKey must be non-empty. When IncAddress is non-empty, the old shielding procedure is
// employed.
//	- If OTDepositPubKey is non-empty, Receivers and Signatures must be non-empty. Each signature must be a valid signature against
// the OTDepositPubKey and the corresponding otaReceiver.
//	- The BTCAddress must be a valid multi-sig address generated based on either IncAddress or OTDepositPubKey.
type APIAddPortalShieldingRequestParams struct {
	// IncAddress is an Incognito payment address associated with the shielding request. It is used in the
	// old shielding procedure in which BTCAddress is repeated for every shielding request of the same IncAddress.
	// Either IncAddress or OTDepositPubKey must be non-empty.
	IncAddress string `json:"IncAddress,omitempty" form:"IncAddress,omitempty"`

	// OTDepositPubKey is a one-time depositing public key associated with the shielding request. It is used to replace
	// the old shielding procedure and provides better privacy level.
	// Either IncAddress or OTDepositPubKey must be non-empty.
	OTDepositPubKey string `json:"OTDepositPubKey,omitempty" form:"OTDepositPubKey,omitempty"`

	// Receivers is a list of OTAReceivers for receiving the shielding assets.
	Receivers []string `json:"Receivers,omitempty" form:"Receivers,omitempty"`

	// Signatures is a list of valid signatures signed on each OTAReceiver against the OTDepositPubKey.
	Signatures []string `json:"Signatures,omitempty" form:"Signatures,omitempty"`

	// BTCAddress is the multi-sig address for receiving public token. It is generated based on either IncAddress or
	// OTDepositPubKey.
	BTCAddress string `json:"BTCAddress" form:"BTCAddress"`
}

func (p APIAddPortalShieldingRequestParams) IsValid() (bool, error) {
	if p.IncAddress == "" && p.OTDepositPubKey == "" {
		return false, fmt.Errorf("either `IncAddress` or `OTDepositPubKey` must be non-empty")
	}

	if p.IncAddress != "" && p.OTDepositPubKey != "" {
		return false, fmt.Errorf("either `IncAddress` or `OTDepositPubKey` must be empty")
	}

	chainCode := p.IncAddress
	if p.OTDepositPubKey != "" {
		chainCode = p.OTDepositPubKey
		if len(p.Receivers) == 0 {
			return false, fmt.Errorf("`Receivers` must be supplied")
		}
		if len(p.Receivers) != len(p.Signatures) {
			return false, fmt.Errorf("expect number of `Signature`'s %v, got %v", len(p.Receivers), len(p.Signatures))
		}

		depositPubKeyBytes, _, err := base58.Base58Check{}.Decode(p.OTDepositPubKey)
		if err != nil {
			return false, fmt.Errorf("`OTDepositPubKey` is invalid")
		}
		depositPubKey, err := new(crypto.Point).FromBytesS(depositPubKeyBytes)
		if err != nil {
			return false, fmt.Errorf("`OTDepositPubKey` is invalid")
		}
		schnorrPubKey := new(privacy.SchnorrPublicKey)
		schnorrPubKey.Set(depositPubKey)

		for i, otaReceiverStr := range p.Receivers {
			sigStr := p.Signatures[i]

			otaReceiver := new(coin.OTAReceiver)
			err = otaReceiver.FromString(otaReceiverStr)
			if err != nil {
				return false, fmt.Errorf("invalid receiver %v", otaReceiverStr)
			}
			if incClient.HasOTAPubKey(base58.Base58Check{}.Encode(otaReceiver.PublicKey.ToBytesS(), 0)) {
				return false, fmt.Errorf("otaKey %v existed", otaReceiverStr)
			}

			sigBytes, _, err := base58.Base58Check{}.Decode(sigStr)
			if err != nil {
				return false, fmt.Errorf("invalid signature %v", sigStr)
			}

			schnorrSig := new(privacy.SchnorrSignature)
			err = schnorrSig.SetBytes(sigBytes)
			if err != nil {
				return false, fmt.Errorf("invalid signature %v", sigStr)
			}

			if !schnorrPubKey.Verify(schnorrSig, common.HashB(otaReceiver.Bytes())) {
				return false, fmt.Errorf("invalid receivingInfo %v: %v", otaReceiverStr, sigStr)
			}
		}
	}

	btcAddr, err := incClient.GeneratePortalShieldingAddress(chainCode, BTCTokenID)
	if err != nil {
		return false, fmt.Errorf("error when generating portal shielding address: %v", err)
	}
	if btcAddr != p.BTCAddress {
		return false, fmt.Errorf("invalid `BTCAddress` for chainCode %v", chainCode)
	}

	return true, nil
}

// APIGetShieldHistoryParams represents a list of parameters for the `getshieldhistory` api.
// A valid request must satisfy the condition in which either IncAddress or OTDepositPubKeys must be non-empty.
type APIGetShieldHistoryParams struct {
	// IncAddress is an Incognito payment address associated with the shielding request. It is used in the
	// old shielding procedure in which BTCAddress is repeated for every shielding request of the same IncAddress.
	// Either IncAddress or OTDepositPubKey must be non-empty.
	IncAddress string `json:"IncAddress,omitempty" form:"IncAddress,omitempty"`

	// OTDepositPubKeys is a list of one-time depositing public keys.
	OTDepositPubKeys []string `json:"OTDepositPubKeys,omitempty" form:"OTDepositPubKeys,omitempty"`

	// TokenID is the ID of the token in need of retrieval.
	TokenID string `json:"TokenID" form:"TokenID"`
}

func (p APIGetShieldHistoryParams) IsValid() (bool, error) {
	if p.IncAddress == "" && len(p.OTDepositPubKeys) == 0 {
		return false, fmt.Errorf("either `incaddress` or `depositpubkeys` must be supplied")
	}

	return true, nil
}

// APICheckPortalShieldAddressExistedParams represents a list of parameters for the `checkportalshieldingaddressexisted` api.
// A valid request must satisfy the condition in which either IncAddress or OTDepositPubKeys must be non-empty.
type APICheckPortalShieldAddressExistedParams struct {
	// IncAddress is an Incognito payment address associated with the shielding request. It is used in the
	// old shielding procedure in which BTCAddress is repeated for every shielding request of the same IncAddress.
	// Either IncAddress or OTDepositPubKey must be non-empty.
	IncAddress string `json:"incaddress,omitempty" form:"incaddress,omitempty"`

	// OTDepositPubKeys is a list of one-time depositing public keys.
	OTDepositPubKey string `json:"depositpubkey,omitempty" form:"depositpubkey,omitempty"`

	// BTCAddress is the multi-sig address for receiving public token. It is generated based on either IncAddress or
	// OTDepositPubKey.
	BTCAddress string `json:"btcaddress" form:"btcaddress"`
}

func (p APICheckPortalShieldAddressExistedParams) IsValid() (bool, error) {
	if p.IncAddress == "" && p.OTDepositPubKey == "" {
		return false, fmt.Errorf("either `incaddress` or `depositpubkey` must be supplied")
	}
	if p.BTCAddress == "" {
		return false, fmt.Errorf("`btcaddress` must be supplied")
	}

	return true, nil
}

// APIGetListPortalShieldingAddressParams represents a list of parameters for the `getlistportalshieldingaddress` api.
type APIGetListPortalShieldingAddressParams struct {
	From int64 `json:"from" form:"from"`
	To   int64 `json:"to" form:"to"`
}

func (p APIGetListPortalShieldingAddressParams) IsValid() (bool, error) {
	if p.To == 0 || p.From > p.To {
		return false, fmt.Errorf("invalid interval")
	}

	return true, nil
}

type APIResponse struct {
	Result interface{}
	Error  *string
}
