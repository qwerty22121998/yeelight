package yeelight

type Method string

const (
	Props          Method = "props"
	GetProp        Method = "get_prop"
	SetCtAbx       Method = "set_ct_abx"
	SetRGB         Method = "set_rgb"
	SetHSV         Method = "set_hsv"
	SetBright      Method = "set_bright"
	SetPower       Method = "set_power"
	Toggle         Method = "toggle"
	SetDefault     Method = "set_default"
	StartCf        Method = "start_cf"
	StopCf         Method = "stop_cf"
	SetScene       Method = "set_scene"
	CronAdd        Method = "cron_add"
	CronGet        Method = "cron_get"
	CronDel        Method = "cron_del"
	SetAdjust      Method = "set_adjust"
	SetMusic       Method = "set_music"
	SetName        Method = "set_name"
	BgSetRGB       Method = "bg_set_rgb"
	BgSetHSV       Method = "bg_set_hsv"
	BgSetCtAbx     Method = "bg_set_ct_abx"
	BgStartCf      Method = "bg_start_cf"
	BgStopCf       Method = "bg_stop_cf"
	BgSetScene     Method = "bg_set_scene"
	BgSetDefault   Method = "bg_set_default"
	BgSetPower     Method = "bg_set_power"
	BgSetBright    Method = "bg_set_bright"
	BgSetAdjust    Method = "bg_set_adjust"
	BgToggle       Method = "bg_toggle"
	DevToggle      Method = "dev_toggle"
	AdjustBright   Method = "adjust_bright"
	AdjustCt       Method = "adjust_ct"
	AdjustColor    Method = "adjust_color"
	BgAdjustBright Method = "bg_adjust_bright"
	BgAdjustCt     Method = "bg_adjust_ct"
	BgAdjustColor  Method = "bg_adjust_color"
)

type Effect string

const (
	EffectSmooth Effect = "smooth"
	EffectSudden Effect = "sudden"
)

func (e Effect) String() string {
	return string(e)
}

type Property string

const (
	Power        Property = "power"
	MainPower    Property = "main_power"
	Bright       Property = "bright"
	Ct           Property = "ct"
	RGB          Property = "rgb"
	Hue          Property = "hue"
	Sat          Property = "sat"
	ColorMode    Property = "color_mode"
	Flowing      Property = "flowing"
	DelayOff     Property = "delayoff"
	FlowParams   Property = "flow_params"
	MusicOn      Property = "music_on"
	Name         Property = "name"
	BgPower      Property = "bg_power"
	BgFlowing    Property = "bg_flowing"
	BgFlowParams Property = "bg_flow_params"
	BgCt         Property = "bg_ct"
	BgLMode      Property = "bg_lmode"
	BgBright     Property = "bg_bright"
	BgRGB        Property = "bg_rgb"
	BgHue        Property = "bg_hue"
	BgSat        Property = "bg_sat"
	NlBr         Property = "nl_br"
	ActiveMode   Property = "active_mode"
)

var AllProperties = []Property{
	Power,
	MainPower,
	Bright,
	Ct,
	RGB,
	Hue,
	Sat,
	ColorMode,
	Flowing,
	DelayOff,
	FlowParams,
	MusicOn,
	Name,
	BgPower,
	BgFlowing,
	BgFlowParams,
	BgCt,
	BgLMode,
	BgBright,
	BgRGB,
	BgHue,
	BgSat,
	NlBr,
	ActiveMode,
}

type Properties map[Property]any
