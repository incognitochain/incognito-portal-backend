package main

import (
	"time"

	"github.com/kamva/mgm/v3"
)

// PortalShieldingData is a record in the MongoDB representing a shielding request.
type PortalShieldingData struct {
	mgm.DefaultModel `bson:",inline"`

	// IncAddress is an Incognito payment address associated with the shielding request. It is used in the
	// old shielding procedure in which BTCAddress is repeated for every shielding request of the same IncAddress.
	IncAddress string `json:"incaddress,omitempty" bson:"incaddress,omitempty"`

	// OTDepositPubKey is a one-time depositing public key associated with the shielding request. It is used to replace
	// the old shielding procedure and provides better privacy level.
	// Either IncAddress or OTDepositPubKey must be non-empty.
	OTDepositPubKey string `json:"depositkey,omitempty" bson:"depositkey,omitempty"`

	// Receivers is a list of OTAReceivers for receiving the shielding assets.
	// It is only used with OTDepositPubKey.
	Receivers []string `json:"receivers,omitempty" bson:"receivers,omitempty"`

	// Signatures is a list of valid signatures signed on each OTAReceiver against the OTDepositPubKey.
	// It is only used with OTDepositPubKey.
	Signatures []string `json:"signatures,omitempty" bson:"signatures,omitempty"`

	// BTCAddress is the multi-sig address for receiving public token. It is generated based on either IncAddress or
	// OTDepositPubKey.
	BTCAddress string `json:"btcaddress" bson:"btcaddress"`

	// TimeStamp is the initializing time of the request.
	TimeStamp int64 `json:"timestamp" bson:"timestamp"`
}

func NewPortalAddressData(requestInfo APIAddPortalShieldingRequestParams) *PortalShieldingData {
	timestamp := time.Now().Unix()
	return &PortalShieldingData{
		IncAddress:      requestInfo.IncAddress,
		OTDepositPubKey: requestInfo.OTDepositPubKey,
		Receivers:       requestInfo.Receivers,
		Signatures:      requestInfo.Signatures,
		BTCAddress:      requestInfo.BTCAddress,
		TimeStamp:       timestamp,
	}
}

func (model *PortalShieldingData) Creating() error {
	curTime := time.Now().UTC()
	model.DefaultModel.DateFields.CreatedAt = curTime
	model.DefaultModel.DateFields.UpdatedAt = curTime
	return nil
}

func (model *PortalShieldingData) Saving() error {
	// Call the DefaultModel Creating hook
	if err := model.DefaultModel.Saving(); err != nil {
		return err
	}

	return nil
}
