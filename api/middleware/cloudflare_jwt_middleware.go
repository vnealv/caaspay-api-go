package middleware

import (
	"caaspay-api-go/api/config"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// Struct to store JWKS data
type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// Cache for JWKS to avoid repeated fetching
var (
	cachedKeys    = map[string]*rsa.PublicKey{}
	cacheMutex    sync.RWMutex
	lastFetchTime time.Time
)

// CloudflareJWTMiddleware validates tokens issued by Cloudflare using JWKS
func CloudflareJWTMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("CF-Access-JWT-Assertion")

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Cloudflare JWT missing"})
			c.Abort()
			return
		}

		// Parse and validate the JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if token.Method != jwt.SigningMethodRS256 {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Method.Alg())
			}
			kid, ok := token.Header["kid"].(string)
			if !ok {
				return nil, errors.New("missing kid in token header")
			}

			// Fetch public key for the given kid
			return fetchJWKSKey(cfg, kid)
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Cloudflare JWT"})
			c.Abort()
			return
		}

		// If the token is valid, continue processing
		c.Next()
	}
}

// fetchJWKSKey fetches the RSA public key for a given kid from JWKS
func fetchJWKSKey(cfg *config.Config, kid string) (*rsa.PublicKey, error) {
	cacheMutex.RLock()
	if key, found := cachedKeys[kid]; found && time.Since(lastFetchTime) < cfg.JWTCloudflare.CacheDuration {
		cacheMutex.RUnlock()
		return key, nil
	}
	cacheMutex.RUnlock()

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Refresh JWKS if needed
	if time.Since(lastFetchTime) >= cfg.JWTCloudflare.CacheDuration {
		if err := updateJWKSCache(cfg.JWTCloudflare.PublicKeyURL); err != nil {
			return nil, err
		}
		lastFetchTime = time.Now()
	}

	key, found := cachedKeys[kid]
	if !found {
		return nil, fmt.Errorf("key with kid %s not found in JWKS", kid)
	}
	return key, nil
}

// updateJWKSCache fetches the latest JWKS and updates the cache
func updateJWKSCache(jwksURL string) error {
	resp, err := http.Get(jwksURL)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	newKeys := map[string]*rsa.PublicKey{}
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" {
			continue // Skip non-RSA keys
		}
		rsaKey, err := parseRSAPublicKeyFromJWK(jwk)
		if err != nil {
			return fmt.Errorf("failed to parse RSA key: %w", err)
		}
		newKeys[jwk.Kid] = rsaKey
	}

	cachedKeys = newKeys
	return nil
}

// parseRSAPublicKeyFromJWK parses a JWK into an RSA public key
func parseRSAPublicKeyFromJWK(jwk JWK) (*rsa.PublicKey, error) {
	nBytes, err := jwt.DecodeSegment(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode N: %w", err)
	}
	eBytes, err := jwt.DecodeSegment(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode E: %w", err)
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	rsaKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}
	return rsaKey, nil
}
