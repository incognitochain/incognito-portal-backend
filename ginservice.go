package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/kamva/mgm/v3"
	stats "github.com/semihalev/gin-stats"
)

func startGinService() {
	logger.Println("initiating api-service...")

	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(stats.RequestStats())

	r.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, stats.Report())
	})
	r.GET("/health", API_HealthCheck)
	r.GET("/checkportalshieldingaddressexisted", API_CheckPortalShieldingAddressExisted)
	r.POST("/addportalshieldingaddress", API_AddPortalShieldingAddress)
	r.GET("/getlistportalshieldingaddress", API_GetListPortalShieldingAddress)
	r.GET("/getestimatedunshieldingfee", API_GetEstimatedUnshieldingFee)
	r.POST("/getshieldhistory", API_GetShieldHistory)
	r.GET("/getshieldhistorybyexternaltxid", API_GetShieldHistoryByExternalTxID)
	err := r.Run("127.0.0.1:" + strconv.Itoa(serviceCfg.APIPort))
	if err != nil {
		panic(err)
	}
}

func API_CheckPortalShieldingAddressExisted(c *gin.Context) {
	prefix := "[CheckShieldAddressExisted]"
	var req APICheckPortalShieldAddressExistedParams
	err := c.BindQuery(&req)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	if isValid, err := req.IsValid(); !isValid || err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	chainCode := req.OTDepositPubKey
	if req.IncAddress != "" {
		chainCode = req.IncAddress
	}

	// check unique
	exists, err := DBCheckPortalShieldDataExisted(chainCode, req.BTCAddress, chainCode == req.IncAddress)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Result: exists,
		Error:  nil,
	})
}

func API_AddPortalShieldingAddress(c *gin.Context) {
	prefix := "[AddShieldAddress]"
	var req APIAddPortalShieldingRequestParams
	err := c.ShouldBindJSON(&req)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	if isValid, err := req.IsValid(); !isValid || err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	chainCode := req.OTDepositPubKey
	if req.IncAddress != "" {
		chainCode = req.IncAddress
	}

	// check unique
	exists, err := DBCheckPortalShieldDataExisted(chainCode, req.BTCAddress, chainCode == req.IncAddress)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}
	if exists {
		msg := "record has already been inserted"
		c.JSON(http.StatusOK, APIResponse{
			Result: nil,
			Error:  &msg,
		})
		return
	}

	err = importBTCAddressToFullNode(req.BTCAddress)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	item := NewPortalAddressData(req)
	err = DBSavePortalShieldingData(*item)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Result: true,
		Error:  nil,
	})
}

func API_GetListPortalShieldingAddress(c *gin.Context) {
	prefix := "[ListPortalShield]"
	var req APIGetListPortalShieldingAddressParams
	err := c.BindQuery(&req)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(fmt.Errorf("invalid parameters")))
		return
	}
	if isValid, err := req.IsValid(); !isValid || err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	list, err := DBGetPortalAddressesByTimestamp(req.From, req.To)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	c.JSON(http.StatusOK, APIResponse{
		Result: list,
		Error:  nil,
	})
}

func API_GetEstimatedUnshieldingFee(c *gin.Context) {
	vBytePerInput := 192.25
	vBytePerOutput := 43.0
	vByteOverhead := 10.75

	feePerVByte, err := getBitcoinFee()
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(fmt.Errorf("could not get bitcoin fee, error: %v", err)))
		return
	}
	estimatedFee := feePerVByte * (2.0*vBytePerInput + 2.0*vBytePerOutput + vByteOverhead)
	estimatedFee *= 1.15 // overpay

	c.JSON(http.StatusOK, APIResponse{
		Result: estimatedFee,
		Error:  nil,
	})
}

func API_GetShieldHistory(c *gin.Context) {
	prefix := "[GetShieldHistory]"
	var req APIGetShieldHistoryParams
	err := c.ShouldBindJSON(&req)
	if err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	if isValid, err := req.IsValid(); !isValid || err != nil {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	if req.TokenID != BTCTokenID {
		logger.Println(prefix, err)
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(fmt.Errorf(
			"TokenID is not a portal token %v", req.TokenID)))
		return
	}

	// Handle old shielding procedure.
	if req.IncAddress != "" {
		h, err := getShieldHistoryForKey(req.IncAddress, true)
		if err != nil {
			logger.Println(prefix, err)
			c.JSON(http.StatusInternalServerError, err)
		} else {
			c.JSON(http.StatusOK, APIResponse{
				Result: h,
				Error:  nil,
			})
		}
		return
	}

	// Handle new shielding procedure with OTDepositPubKey's.
	res := make(map[string][]PortalShieldHistory)
	for _, depositPubKey := range req.OTDepositPubKeys {
		h, err := getShieldHistoryForKey(depositPubKey)
		if err != nil {
			logger.Println(prefix, err)
			c.JSON(http.StatusInternalServerError, fmt.Errorf("retrieving shielding history for key %v encountered "+
				"an error: %v", depositPubKey, err))
			return
		}
		res[depositPubKey] = h
	}

	c.JSON(http.StatusOK, APIResponse{
		Result: res,
		Error:  nil,
	})
}

func API_GetShieldHistoryByExternalTxID(c *gin.Context) {
	externalTxID := c.Query("externaltxid")
	tokenID := c.Query("tokenid")
	if tokenID != BTCTokenID {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(fmt.Errorf(
			"TokenID is not a portal token %v", tokenID)))
		return
	}

	txIDHash, err := chainhash.NewHashFromStr(externalTxID)
	if err != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(
			fmt.Errorf("invalid external txID %v - with err: %v", externalTxID, err)))
		return
	}

	res, err := btcClient.GetTransaction(txIDHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(
			fmt.Errorf("could not get external txID %v - with err: %v", externalTxID, err)))
		return
	}

	status := getStatusFromConfirmation(int(res.Confirmations))
	history := PortalShieldHistory{
		ExternalTxID:  externalTxID,
		Status:        status,
		Confirmations: res.Confirmations,
	}
	c.JSON(http.StatusOK, APIResponse{
		Result: history,
		Error:  nil,
	})

}

func API_HealthCheck(c *gin.Context) {
	//ping pong vs mongo
	status := "healthy"
	mongoStatus := "connected"
	btcNodeStatus := "connected"
	_, cd, _, _ := mgm.DefaultConfigs()
	err := cd.Ping(context.Background(), nil)
	if err != nil {
		status = "unhealthy"
		mongoStatus = "disconnected"
	}
	err = btcClient.Ping()
	if err != nil {
		status = "unhealthy"
		btcNodeStatus = "disconnected"
	}
	c.JSON(http.StatusOK, gin.H{
		"status":      status,
		"mongo":       mongoStatus,
		"btcfullnode": btcNodeStatus,
	})
}

func buildGinErrorRespond(err error) *APIResponse {
	errStr := err.Error()
	respond := APIResponse{
		Result: nil,
		Error:  &errStr,
	}
	return &respond
}
