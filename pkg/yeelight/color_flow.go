package yeelight

import (
	"fmt"
	"strings"
)

type FlowAction int

const (
	FlowRecover FlowAction = 0
	FlowStay    FlowAction = 1
	FlowOff     FlowAction = 2
)

type FlowMode int

const (
	FlowRGB   FlowMode = 1
	FlowCT    FlowMode = 2
	FlowSleep FlowMode = 7
)

type FlowExpression struct {
	Duration   int
	Mode       FlowMode
	Value      int
	Brightness int
}

type ColorFlow struct {
	Count      int
	Action     FlowAction
	Expression []FlowExpression
}

func (c ColorFlow) Build() []any {
	res := []any{c.Count, c.Action}
	sb := strings.Builder{}
	for i, exp := range c.Expression {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%d,%d,%d,%d", exp.Duration, exp.Mode, exp.Value, exp.Brightness))
	}
	res = append(res, sb.String())
	return res
}
