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
	BGProact     *int    `json:"bg_proact,omitempty"`
}

func mergeValue[T any](origin **T, new *T) {
	if new != nil {
		*origin = new
	}
}

func mergeData(origin, new *Data) {
	mergeValue(&origin.Power, new.Power)
	mergeValue(&origin.MainPower, new.MainPower)
	mergeValue(&origin.Bright, new.Bright)
	mergeValue(&origin.Ct, new.Ct)
	mergeValue(&origin.RGB, new.RGB)
	mergeValue(&origin.Hue, new.Hue)
	mergeValue(&origin.Sat, new.Sat)
	mergeValue(&origin.ColorMode, new.ColorMode)
	mergeValue(&origin.Flowing, new.Flowing)
	mergeValue(&origin.DelayOff, new.DelayOff)
	mergeValue(&origin.FlowParams, new.FlowParams)
	mergeValue(&origin.MusicOn, new.MusicOn)
	mergeValue(&origin.Name, new.Name)
	mergeValue(&origin.BgPower, new.BgPower)
	mergeValue(&origin.BgFlowing, new.BgFlowing)
	mergeValue(&origin.BgFlowParams, new.BgFlowParams)
	mergeValue(&origin.BgCt, new.BgCt)
	mergeValue(&origin.BgLMode, new.BgLMode)
	mergeValue(&origin.BgBright, new.BgBright)
	mergeValue(&origin.BgRGB, new.BgRGB)
	mergeValue(&origin.BgSat, new.BgSat)
	mergeValue(&origin.NLBr, new.NLBr)
	mergeValue(&origin.ActiveMode, new.ActiveMode)
	mergeValue(&origin.BGProact, new.BGProact)
}

type Info struct {
	IP        string          `json:"ip"`
	Name      string          `json:"name,omitempty"`
	ID        string          `json:"id,omitempty"`
	Model     string          `json:"model,omitempty"`
	FWVersion string          `json:"fw_ver,omitempty"`
	Methods   map[Method]bool `json:"methods,omitempty"`
}
