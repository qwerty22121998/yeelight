package yeelight

import (
	"fmt"
)

// Command represents command sent to the Yeelight device. Use C() to create a Command instance.
type Command struct {
	ID     int    `json:"id"`
	Method Method `json:"method"`
	Params any    `json:"params"`
}

func C(method Method, params ...any) Command {
	if len(params) == 0 {
		params = []any{}
	}
	return Command{
		Method: method,
		Params: params,
	}
}

type Response struct {
	ID     int      `json:"id"`
	Method Method   `json:"method"`
	Params Data     `json:"params"`
	Result []string `json:"result"`
	Error  *Error   `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.Code, e.Message)
}
