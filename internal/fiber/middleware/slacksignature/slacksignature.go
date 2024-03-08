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

func abs(n int64) int64 {
	y := n >> 63
	return (n ^ y) - y
}

func errorHandler(c fiber.Ctx, err error) error {
	fmt.Printf("errorHandler %v\n", err)
	return c.Status(fiber.StatusNotFound).SendString("")
}

// https://api.slack.com/authentication/verifying-requests-from-slack#making__validating-a-request
func New(key string) fiber.Handler {
	if key == "" {
		panic("SLACK_SIGNING_SECRET not set")
	}

	return func(c fiber.Ctx) error {
		tsStr := string(c.Request().Header.Peek("x-slack-request-timestamp"))
		if tsStr == "" {
			return errorHandler(c, fmt.Errorf("timestamp is missing"))
		}

		tsInt, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			return errorHandler(c, fmt.Errorf("invalid timestamp '%s': %w", tsStr, err))
		}

		// tolerance of 5 minutes
		if abs(time.Now().Unix()-tsInt) > 60*5 {
			return errorHandler(c, fmt.Errorf("timestamp too old: %d", tsInt))
		}

		signature := string(c.Request().Header.Peek("x-slack-signature"))
		if signature == "" {
			return errorHandler(c, fmt.Errorf("signature is missing"))
		}

		body := string(c.Request().Body())

		hmac := hmac.New(sha256.New, []byte(key))
		hmac.Write([]byte("v0:" + tsStr + ":" + body))
		hmacHex := "v0=" + hex.EncodeToString(hmac.Sum(nil))

		if hmacHex != signature {
			return errorHandler(c, fmt.Errorf("signature mismatch '%s' != '%s'", hmacHex, signature))
		}

		return c.Next()
	}
}
