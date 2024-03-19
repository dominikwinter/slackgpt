package slacksignature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Config struct {
	Key             string
	Tolerance       time.Duration
	HeaderTimestamp string
	HeaderSignature string
}

var ConfigDefault = Config{
	Key:             "",
	Tolerance:       5 * time.Minute,
	HeaderTimestamp: "x-slack-request-timestamp",
	HeaderSignature: "x-slack-signature",
}

func configDefault(config ...Config) Config {
	if len(config) < 1 {
		return ConfigDefault
	}

	cfg := config[0]

	return cfg
}

func abs(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

func errorHandler(c fiber.Ctx, err error) error {
	fmt.Printf("errorHandler %v\n", err)
	return c.Status(fiber.StatusNotFound).SendString("")
}

// https://api.slack.com/authentication/verifying-requests-from-slack#making__validating-a-request
func New(config ...Config) fiber.Handler {
	cfg := configDefault(config...)

	if cfg.Key == "" {
		panic("slack signing secret not set")
	}

	tolerance := int64(cfg.Tolerance.Seconds())

	return func(c fiber.Ctx) error {
		tsStr := string(c.Request().Header.Peek(cfg.HeaderTimestamp))
		if tsStr == "" {
			return errorHandler(c, fmt.Errorf("timestamp is missing"))
		}

		tsInt, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return errorHandler(c, fmt.Errorf("invalid timestamp '%s': %w", tsStr, err))
		}

		if abs(time.Now().Unix()-tsInt) > tolerance {
			return errorHandler(c, fmt.Errorf("timestamp too old: %d", tsInt))
		}

		signature := string(c.Request().Header.Peek(cfg.HeaderSignature))
		if signature == "" {
			return errorHandler(c, fmt.Errorf("signature is missing"))
		}

		body := string(c.Request().Body())

		hmac := hmac.New(sha256.New, []byte(cfg.Key))
		hmac.Write([]byte("v0:" + tsStr + ":" + body))
		hmacHex := "v0=" + hex.EncodeToString(hmac.Sum(nil))

		if hmacHex != signature {
			return errorHandler(c, fmt.Errorf("signature mismatch '%s' != '%s'", hmacHex, signature))
		}

		return c.Next()
	}
}
