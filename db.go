package main

import (
	"context"
	"time"

	"github.com/kamva/mgm/v3"
	"github.com/kamva/mgm/v3/operator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

func connectDB() error {
	err := mgm.SetDefaultConfig(nil, serviceCfg.MongoDB, options.Client().ApplyURI(serviceCfg.MongoAddress))
	if err != nil {
		return err
	}
	_, cd, _, _ := mgm.DefaultConfigs()
	err = cd.Ping(context.Background(), nil)
	if err != nil {
		return err
	}
	logger.Println("Database Connected!")
	return nil
}

func DBDropPortalAddressIndices() error {
	startTime := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(5)*DB_OPERATION_TIMEOUT)

	_, err := mgm.Coll(&PortalShieldingData{}).Indexes().DropAll(ctx)
	if err != nil {
		logger.Printf("failed to drop all indices of index portal addresses\n")
		return err
	}
	logger.Printf("Dropping indices succeeded after %v", time.Since(startTime))
	return nil
}

func DBCreatePortalAddressIndex() error {
	startTime := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(5)*DB_OPERATION_TIMEOUT)

	coinMdl := []mongo.IndexModel{
		{
			Keys:    bsonx.Doc{{Key: "incaddress", Value: bsonx.Int32(1)}, {Key: "depositkey", Value: bsonx.Int32(1)}, {Key: "btcaddress", Value: bsonx.Int32(1)}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bsonx.Doc{{Key: "timestamp", Value: bsonx.Int32(1)}},
		},
	}
	_, err := mgm.Coll(&PortalShieldingData{}).Indexes().CreateMany(ctx, coinMdl)
	if err != nil {
		logger.Printf("failed to index portal addresses in %v", time.Since(startTime))
		return err
	}

	logger.Printf("success index portal addresses in %v", time.Since(startTime))
	return nil
}

func DBCheckPortalShieldDataExisted(chainCode, btcAddress string, usePaymentAddress ...bool) (bool, error) {
	startTime := time.Now()

	filter := bson.M{"depositkey": bson.M{operator.Eq: chainCode}, "btcaddress": bson.M{operator.Eq: btcAddress}}
	if len(usePaymentAddress) > 0 && usePaymentAddress[0] {
		filter = bson.M{"incaddress": bson.M{operator.Eq: chainCode}, "btcaddress": bson.M{operator.Eq: btcAddress}}
	}
	logger.Printf("filter: %v\n", filter)

	var result PortalShieldingData
	err := mgm.Coll(&PortalShieldingData{}).First(filter, &result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			logger.Printf("check portal address not existed in %v", time.Since(startTime))
			return false, nil
		}
		return false, err
	}
	logger.Printf("check portal address existed in %v", time.Since(startTime))
	return true, nil
}

func DBSavePortalShieldingData(item PortalShieldingData) error {
	startTime := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(5)*DB_OPERATION_TIMEOUT)

	err := item.Creating()
	if err != nil {
		return err
	}
	_, err = mgm.Coll(&PortalShieldingData{}).InsertOne(ctx, item)
	if err != nil {
		logger.Printf("failed to insert portal address %v in %v", item, time.Since(startTime))
		return err
	}

	logger.Printf("inserted portal address %v in %v", item, time.Since(startTime))
	return nil
}

func DBGetPortalAddressesByTimestamp(fromTimeStamp int64, toTimeStamp int64) ([]PortalShieldingData, error) {
	startTime := time.Now()
	list := make([]PortalShieldingData, 0)
	filter := bson.M{"timestamp": bson.M{operator.Gte: fromTimeStamp, operator.Lt: toTimeStamp}}

	err := mgm.Coll(&PortalShieldingData{}).SimpleFind(&list, filter)
	if err != nil {
		return nil, err
	}
	logger.Printf("found %v records in %v", len(list), time.Since(startTime))

	return list, nil
}

func DBGetBTCAddressByChainCode(chainCode string, usePaymentAddress ...bool) (string, error) {
	startTime := time.Now()

	filter := bson.M{"depositkey": bson.M{operator.Eq: chainCode}}
	if len(usePaymentAddress) > 0 && usePaymentAddress[0] {
		filter = bson.M{"incaddress": bson.M{operator.Eq: chainCode}}
	}
	var result PortalShieldingData
	err := mgm.Coll(&PortalShieldingData{}).First(filter, &result)
	if err != nil {
		return "", err
	}
	logger.Printf("get btc address by chainCode in %v", time.Since(startTime))
	return result.BTCAddress, nil
}
