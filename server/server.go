// DONOTCOVER

package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echolog "github.com/labstack/gommon/log"

	"github.com/vitwit/resolute/server/clients"
	"github.com/vitwit/resolute/server/config"
	"github.com/vitwit/resolute/server/cron"
	"github.com/vitwit/resolute/server/handler"
	middle "github.com/vitwit/resolute/server/middleware"
	"github.com/vitwit/resolute/server/model"

	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func init() {
	config, err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}
	// Initialize the Redis client
	clients.InitializeRedis(config.REDIS_URI, "", 0)
	
	// Load chains from networks.json into Redis immediately on startup
	log.Println("Loading chain configurations from networks.json into Redis...")
	loadChainsIntoRedis()
}

// loadChainsIntoRedis loads chain configurations from networks.json into Redis
func loadChainsIntoRedis() {
	data := config.GetChainAPIs()
	if data == nil {
		log.Println("Warning: No chain data found in networks.json")
		return
	}
	
	// Set default RestURI for each chain (taking the first one from RestURIs)
	for _, c := range data {
		if len(c.RestURIs) > 0 && c.RestURI == "" {
			c.RestURI = c.RestURIs[0]
		}
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling chain data: %v", err)
		return
	}

	err = clients.SetValue("chains", string(bytes))
	if err != nil {
		log.Printf("Error storing chains in Redis: %v", err)
		return
	}
	
	log.Printf("Successfully loaded %d chains into Redis", len(data))
}

func main() {
	e := echo.New()
	e.Logger.SetLevel(echolog.INFO)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	config, err := config.ParseConfig()
	if err != nil {
		log.Fatal(err)
	}

	cfg := config.DB
	apiCfg := config.API

	// TODO: add ssl support
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DatabaseName)

	// open database
	db, err := sql.Open("postgres", psqlconn)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)

	// check db
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Initialize handler
	h := &handler.Handler{DB: db}
	m := &middle.Handler{DB: db}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))

	// Routes
	e.POST("/multisig", h.CreateMultisigAccount, m.AuthMiddleware)
	e.GET("/multisig/accounts/:address", h.GetMultisigAccounts)
	e.GET("/multisig/:address", h.GetMultisigAccount)
	e.DELETE("/multisig/:address", h.DeleteMultisigAccount, m.AuthMiddleware, m.IsMultisigAdmin)
	e.POST("/multisig/:address/tx", h.CreateTransaction, m.AuthMiddleware, m.IsMultisigMember)
	e.GET("/multisig/:address/tx/:id", h.GetTransaction)
	e.POST("/multisig/:address/tx/:id", h.UpdateTransactionInfo, m.AuthMiddleware, m.IsMultisigMember)
	e.DELETE("/multisig/:address/tx/:id", h.DeleteTransaction, m.AuthMiddleware, m.IsMultisigAdmin)
	e.POST("/multisig/:address/sign-tx/:id", h.SignTransaction, m.AuthMiddleware, m.IsMultisigMember)
	e.GET("/multisig/:address/txs", h.GetTransactions)
	e.GET("/accounts/:address/all-txns", h.GetAllMultisigTxns)
	e.POST("/transactions", h.GetRecentTransactions)
	e.GET("/txns/:chainId/:address", h.GetAllTransactions)
	e.GET("/txns/:chainId/:address/:txhash", h.GetChainTxHash)
	e.GET("/search/txns/:txhash", h.GetTxHash)

	// users
	e.POST("/users/:address/signature", h.CreateUserSignature)
	e.GET("/users/:address", h.GetUser)

	e.GET("/tokens-info", h.GetTokensInfo)
	e.GET("/tokens-info/:denom", h.GetTokenInfo)

	// Debug endpoint to list available chains
	e.GET("/debug/chains", func(c echo.Context) error {
		chains := clients.GetChains()
		if chains == nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to get chains from Redis",
			})
		}
		
		chainInfo := make([]map[string]interface{}, len(chains))
		for i, chain := range chains {
			chainInfo[i] = map[string]interface{}{
				"chainId":     chain.ChainId,
				"restURI":     chain.RestURI,
				"rpcURI":      chain.RpcURI,
				"sourceEnd":   chain.SourceEnd,
				"checkStatus": chain.CheckStatus,
			}
		}
		
		return c.JSON(http.StatusOK, map[string]interface{}{
			"total_chains": len(chains),
			"chains": chainInfo,
		})
	})

	e.POST("/cosmos/tx/v1beta1/txs", proxyHandler1)

	e.Any("/*", proxyHandler)

	e.GET("/", func(c echo.Context) error {

		return c.JSON(http.StatusOK, model.SuccessResponse{
			Status:  "success",
			Message: "server up",
		})
	})
	e.RouteNotFound("*", func(c echo.Context) error {
		return c.JSON(http.StatusOK, model.ErrorResponse{
			Status:  "error",
			Message: "route not found",
		})
	})

	// Setup coingecko cron job
	cronClient := cron.NewCron(config, db)
	cronClient.Start()

	// Start server
	// TODO: add ip and port
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", apiCfg.Port)))
}

func proxyHandler1(c echo.Context) error {
	config, err := config.ParseConfig()
	if err != nil {
		log.Printf("Failed to parse config in proxyHandler1: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Configuration error",
			"details": err.Error(),
		})
	}

	type RequestBody struct {
		Mode    string `json:"mode"`
		TxBytes string `json:"tx_bytes"`
	}

	reqBody := new(RequestBody)

	// Bind the request body to the struct
	if err := c.Bind(reqBody); err != nil {
		log.Printf("Failed to bind request body in proxyHandler1: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
			"details": err.Error(),
		})
	}

	// Convert the struct to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Failed to marshal request body in proxyHandler1: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Error encoding JSON",
			"details": err.Error(),
		})
	}

	chainId := c.QueryParam("chain")
	log.Printf("Looking up chain details for chainId: %s", chainId)
	
	chanDetails := clients.GetChain(chainId)
	if chanDetails == nil {
		log.Printf("Failed to get chain details for chainId: %s", chainId)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Chain not found",
			"details": fmt.Sprintf("No configuration found for chain: %s", chainId),
		})
	}

	// URL to which the POST request will be sent
	targetURL := chanDetails.RestURI + "/cosmos/tx/v1beta1/txs"
	log.Printf("Proxying request to: %s", targetURL)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create HTTP request to %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Error creating request",
			"details": err.Error(),
		})
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")

	if chanDetails.SourceEnd == "mintscan" {
		authorizationToken := fmt.Sprintf("Bearer %s", config.MINTSCAN_TOKEN.Token)
		req.Header.Add("Authorization", authorizationToken)
		log.Printf("Added Mintscan authorization for chain %s", chainId)
	}

	if chanDetails.SourceEnd == "numia" {
		bearerToken := config.NUMIA_BEARER_TOKEN.Token
		var authorization = "Bearer " + bearerToken
		req.Header.Add("Authorization", authorization)
		log.Printf("Added Numia authorization for chain %s", chainId)
	}

	// Create a new HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send HTTP request to %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Error sending request",
			"details": err.Error(),
			"target_url": targetURL,
		})
	}
	defer resp.Body.Close()

	log.Printf("Received response from %s with status: %d", targetURL, resp.StatusCode)

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body from %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Error reading response",
			"details": err.Error(),
		})
	}

	// If the upstream service returned an error, log it and pass it through
	if resp.StatusCode >= 400 {
		log.Printf("Upstream service error from %s (status %d): %s", targetURL, resp.StatusCode, string(body))
	}

	// Respond back to the original request with the same status code from upstream
	return c.JSONBlob(resp.StatusCode, body)
}

func proxyHandler(c echo.Context) error {
	config, err := config.ParseConfig()
	if err != nil {
		log.Printf("Failed to parse config in proxyHandler: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Configuration error",
			"details": err.Error(),
		})
	}

	chainId := c.QueryParam("chain")
	requestPath := c.Request().URL.Path
	requestMethod := c.Request().Method
	
	log.Printf("Proxying %s request to path: %s for chain: %s", requestMethod, requestPath, chainId)
	
	chanDetails := clients.GetChain(chainId)
	if chanDetails == nil {
		log.Printf("Failed to get chain details for chainId: %s", chainId)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Chain not found",
			"details": fmt.Sprintf("No configuration found for chain: %s", chainId),
		})
	}
	
	// Construct the target URL based on the incoming request
	targetBase := chanDetails.RestURI
	targetURL := targetBase + c.Request().URL.Path
	if c.Request().URL.RawQuery != "" {
		targetURL += "?" + c.Request().URL.RawQuery
	}
	
	log.Printf("Proxying to target URL: %s", targetURL)

	// Create a new request to the target URL
	req, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
	if err != nil {
		log.Printf("Failed to create request to %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create request",
			"details": err.Error(),
			"target_url": targetURL,
		})
	}
	
	// Forward headers from the original request
	for name, values := range c.Request().Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	req.Header.Set("Content-Type", "application/json")

	// Add Authorization header
	if chanDetails.SourceEnd == "mintscan" {
		authorizationToken := fmt.Sprintf("Bearer %s", config.MINTSCAN_TOKEN.Token)
		req.Header.Add("Authorization", authorizationToken)
		log.Printf("Added Mintscan authorization for chain %s", chainId)
	}

	if chanDetails.SourceEnd == "numia" {
		bearerToken := config.NUMIA_BEARER_TOKEN.Token
		var authorization = "Bearer " + bearerToken
		req.Header.Add("Authorization", authorization)
		log.Printf("Added Numia authorization for chain %s", chainId)
	}

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to make request to %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to make request",
			"details": err.Error(),
			"target_url": targetURL,
		})
	}
	defer resp.Body.Close()

	log.Printf("Received response from %s with status: %d", targetURL, resp.StatusCode)

	// Check the content encoding and decode accordingly
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("Failed to create gzip reader for response from %s: %v", targetURL, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to decompress response",
				"details": err.Error(),
			})
		}
		defer reader.Close()
	case "br":
		reader = ioutil.NopCloser(brotli.NewReader(resp.Body))
		defer reader.Close()
	default:
		reader = resp.Body
	}

	// Read the decompressed or raw body
	bodyBytes, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Printf("Failed to read response body from %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read response body",
			"details": err.Error(),
		})
	}

	// If the upstream service returned an error, log it
	if resp.StatusCode >= 400 {
		log.Printf("Upstream service error from %s (status %d): %s", targetURL, resp.StatusCode, string(bodyBytes))
	}

	// Set content type and response
	c.Response().Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Response().WriteHeader(resp.StatusCode)
	_, err = c.Response().Writer.Write(bodyBytes)
	if err != nil {
		log.Printf("Failed to write response body from %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to write response body",
			"details": err.Error(),
		})
	}

	return nil
}
