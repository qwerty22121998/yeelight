package yeelight

import (
	"fmt"
)

type Command struct {
	ID     int    `json:"id"`
	Method Method `json:"method"`
	Params []any  `json:"params"`
}

type Response struct {
	ID     int              `json:"id"`
	Method Method           `json:"method"`
	Params map[Property]any `json:"params"`
	Result []any            `json:"result"`
	Error  *Error           `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Message)
}
