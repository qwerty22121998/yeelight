package yeelight

type Data struct {
	Power        *string `json:"power,omitempty"`
	MainPower    *string `json:"main_power,omitempty"`
	Bright       *int    `json:"bright,omitempty"`
	Ct           *int    `json:"ct,omitempty"`
	RGB          *int    `json:"rgb,omitempty"`
	Hue          *int    `json:"hue,omitempty"`
	Sat          *int    `json:"sat,omitempty"`
	ColorMode    *int    `json:"color_mode,omitempty"`
	Flowing      *int    `json:"flowing,omitempty"`
	DelayOff     *int    `json:"delayoff,omitempty"`
	FlowParams   *string `json:"flow_params,omitempty"`
	MusicOn      *int    `json:"music_on,omitempty"`
	Name         *string `json:"name,omitempty"`
	BgPower      *string `json:"bg_power,omitempty"`
	BgFlowing    *int    `json:"bg_flowing,omitempty"`
	BgFlowParams *string `json:"bg_flow_params,omitempty"`
	BgCt         *int    `json:"bg_ct,omitempty"`
	BgLMode      *int    `json:"bg_lmode,omitempty"`
	BgBright     *int    `json:"bg_bright,omitempty"`
	BgRGB        *int    `json:"bg_rgb,omitempty"`
	BgSat        *int    `json:"bg_sat,omitempty"`
	NLBr         *int    `json:"nl_br,omitempty"`
	ActiveMode   *int    `json:"active_mode,omitempty"`
}
type Info struct {
	IP      string          `json:"ip"`
	ID      string          `json:"id,omitempty"`
	Model   string          `json:"model,omitempty"`
	Methods map[Method]bool `json:"methods,omitempty"`
}
