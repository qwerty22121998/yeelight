package yeelight

import (
	"errors"
	"fmt"
	"testing"
)

func TestIsUnsupported(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"device unsupported reply", &Error{Code: -1, Message: "method not supported"}, true},
		{"wrapped unsupported reply", fmt.Errorf("send: %w", &Error{Code: -1, Message: "method not supported"}), true},
		{"other device error", &Error{Code: -5, Message: "general error"}, false},
		{"plain error", errors.New("timeout"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := IsUnsupported(c.err); got != c.want {
			t.Errorf("%s: IsUnsupported = %v, want %v", c.name, got, c.want)
		}
	}
}
