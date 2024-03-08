package helper

import (
	"fmt"
)

type ErrorMessage struct {
	Message string `json:"message"`
}

func (msg *ErrorMessage) Error() string {
	return fmt.Sprintf("API Error: %s", msg.Message)
}
