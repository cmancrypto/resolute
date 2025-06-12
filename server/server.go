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

	// Debug endpoint for transaction testing
	e.POST("/debug/tx", func(c echo.Context) error {
		chainId := c.QueryParam("chain")
		if chainId == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Missing chain parameter",
			})
		}

		// Get chain details
		chanDetails := clients.GetChain(chainId)
		if chanDetails == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Chain not found",
				"details": fmt.Sprintf("No configuration found for chain: %s", chainId),
			})
		}

		// Read and log the request body
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Failed to read request body",
				"details": err.Error(),
			})
		}

		// Parse the transaction request
		var txRequest map[string]interface{}
		if err := json.Unmarshal(body, &txRequest); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid JSON in request body",
				"details": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Transaction debug info",
			"chain_config": map[string]interface{}{
				"chainId":   chanDetails.ChainId,
				"restURI":   chanDetails.RestURI,
				"rpcURI":    chanDetails.RpcURI,
				"sourceEnd": chanDetails.SourceEnd,
			},
			"request_info": map[string]interface{}{
				"body_size":    len(body),
				"content_type": c.Request().Header.Get("Content-Type"),
				"user_agent":   c.Request().Header.Get("User-Agent"),
			},
			"tx_request": txRequest,
			"endpoints": map[string]interface{}{
				"rest_tx_url": chanDetails.RestURI + "/cosmos/tx/v1beta1/txs",
				"rpc_available": chanDetails.RpcURI != "",
				"rpc_broadcast_url": chanDetails.RpcURI + "/broadcast_tx_commit",
			},
		})
	})

	// Debug endpoint to test RPC connectivity
	e.GET("/debug/rpc/:chainId", func(c echo.Context) error {
		chainId := c.Param("chainId")
		
		chanDetails := clients.GetChain(chainId)
		if chanDetails == nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Chain not found",
			})
		}

		result := map[string]interface{}{
			"chainId": chainId,
			"restURI": chanDetails.RestURI,
			"rpcURI":  chanDetails.RpcURI,
		}

		// Test REST endpoint
		if chanDetails.RestURI != "" {
			resp, err := http.Get(chanDetails.RestURI + "/cosmos/base/tendermint/v1beta1/node_info")
			if err != nil {
				result["rest_status"] = "error: " + err.Error()
			} else {
				result["rest_status"] = fmt.Sprintf("status: %d", resp.StatusCode)
				resp.Body.Close()
			}
		}

		// Test RPC endpoint
		if chanDetails.RpcURI != "" {
			resp, err := http.Get(chanDetails.RpcURI + "/status")
			if err != nil {
				result["rpc_status"] = "error: " + err.Error()
			} else {
				result["rpc_status"] = fmt.Sprintf("status: %d", resp.StatusCode)
				resp.Body.Close()
			}
		} else {
			result["rpc_status"] = "no RPC URI configured"
		}

		return c.JSON(http.StatusOK, result)
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

	// Log the request body for debugging
	log.Printf("POST /cosmos/tx/v1beta1/txs - Mode: %s, TxBytes length: %d", reqBody.Mode, len(reqBody.TxBytes))

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

	// URL to which the POST request will be sent (WITHOUT the chain parameter)
	targetURL := chanDetails.RestURI + "/cosmos/tx/v1beta1/txs"
	log.Printf("Proxying POST request to: %s", targetURL)

	// Create a new HTTP request
	req, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create HTTP request to %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Error creating request",
			"details": err.Error(),
		})
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Resolute-Proxy/1.0")

	// Add authorization if needed
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

	log.Printf("Received response from %s with status: %d, content-type: %s", targetURL, resp.StatusCode, resp.Header.Get("Content-Type"))

	// Read the response body (handle compression if needed)
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
		log.Printf("Decompressing gzip response")
	case "br":
		reader = ioutil.NopCloser(brotli.NewReader(resp.Body))
		defer reader.Close()
		log.Printf("Decompressing brotli response")
	default:
		reader = resp.Body
	}

	// Read the response body
	body, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Printf("Failed to read response body from %s: %v", targetURL, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Error reading response",
			"details": err.Error(),
		})
	}

	// Log the response for debugging
	if resp.StatusCode >= 400 {
		log.Printf("Upstream service error from %s (status %d): %s", targetURL, resp.StatusCode, string(body))
	} else {
		log.Printf("Successful response from %s (status %d), body length: %d", targetURL, resp.StatusCode, len(body))
		// Also log the actual response body for debugging transactions
		log.Printf("Response body from %s: %s", targetURL, string(body))
	}

	// Parse and return JSON response properly
	if resp.Header.Get("Content-Type") == "application/json" || resp.Header.Get("Content-Type") == "application/json; charset=utf-8" {
		var jsonResponse interface{}
		if err := json.Unmarshal(body, &jsonResponse); err == nil {
			// Check if it's a transaction response with an error
			if respMap, ok := jsonResponse.(map[string]interface{}); ok {
				// Check for cosmos transaction response patterns
				if code, hasCode := respMap["code"]; hasCode {
					if codeNum, ok := code.(float64); ok && codeNum != 0 {
						log.Printf("Transaction failed with code %v: %v", code, respMap)
					}
				}
				// Check for other error patterns
				if txResponse, hasTxResponse := respMap["tx_response"]; hasTxResponse {
					if trMap, ok := txResponse.(map[string]interface{}); ok {
						if code, hasCode := trMap["code"]; hasCode {
							if codeNum, ok := code.(float64); ok && codeNum != 0 {
								log.Printf("Transaction failed in tx_response with code %v: %v", code, trMap)
							}
						}
					}
				}
			}
			return c.JSON(resp.StatusCode, jsonResponse)
		} else {
			log.Printf("Failed to parse JSON response from %s: %v", targetURL, err)
		}
	}
	
	// Fall back to returning raw response with proper status code
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
	
	// Filter out the 'chain' parameter and preserve other query parameters
	if c.Request().URL.RawQuery != "" {
		query := c.Request().URL.Query()
		query.Del("chain") // Remove the chain parameter
		if len(query) > 0 {
			targetURL += "?" + query.Encode()
		}
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
	
	// Forward headers from the original request, but filter out problematic ones
	for name, values := range c.Request().Header {
		// Skip certain headers that shouldn't be forwarded
		if name == "Host" || name == "Connection" || name == "Accept-Encoding" {
			continue
		}
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}
	
	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Resolute-Proxy/1.0")

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

	log.Printf("Received response from %s with status: %d, content-type: %s", targetURL, resp.StatusCode, resp.Header.Get("Content-Type"))

	// Read the response body (handle compression if needed)
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
		log.Printf("Decompressing gzip response")
	case "br":
		reader = ioutil.NopCloser(brotli.NewReader(resp.Body))
		defer reader.Close()
		log.Printf("Decompressing brotli response")
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

	// Return JSON response properly
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	
	// Set the response headers
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	
	// If it's JSON content, parse and return as JSON to ensure proper formatting
	if resp.Header.Get("Content-Type") == "application/json" || resp.Header.Get("Content-Type") == "application/json; charset=utf-8" {
		var jsonResponse interface{}
		if err := json.Unmarshal(bodyBytes, &jsonResponse); err == nil {
			// Check if it's a transaction response with an error
			if respMap, ok := jsonResponse.(map[string]interface{}); ok {
				// Check for cosmos transaction response patterns
				if code, hasCode := respMap["code"]; hasCode {
					if codeNum, ok := code.(float64); ok && codeNum != 0 {
						log.Printf("Transaction failed with code %v: %v", code, respMap)
					}
				}
				// Check for other error patterns
				if txResponse, hasTxResponse := respMap["tx_response"]; hasTxResponse {
					if trMap, ok := txResponse.(map[string]interface{}); ok {
						if code, hasCode := trMap["code"]; hasCode {
							if codeNum, ok := code.(float64); ok && codeNum != 0 {
								log.Printf("Transaction failed in tx_response with code %v: %v", code, trMap)
							}
						}
					}
				}
			}
			return c.JSON(resp.StatusCode, jsonResponse)
		}
	}
	
	// Fall back to returning raw bytes with proper status code
	return c.JSONBlob(resp.StatusCode, bodyBytes)
}
