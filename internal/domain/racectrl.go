package domain

const (
	RaceCtrlMsgCategoryTrackStatus = "TRACK_STATUS"
	RaceCtrlMsgCategoryFIA         = "FIA"
	RaceCtrlMsgCategoryOther       = "OTHER"
)

const (
	RaceCtrlMsgTitleSC               = "SAFETY\nCAR"
	RaceCtrlMsgTitleVSC              = "VSC"
	RaceCtrlMsgTitleFlagGreen        = "GREEN\nFLAG"
	RaceCtrlMsgTitleFlagBlue         = "BLUE\nFLAG"
	RaceCtrlMsgTitleFlagYellow       = "YELLOW\nFLAG"
	RaceCtrlMsgTitleFlagDoubleYellow = "DOUBLE\nYELLOW"
	RaceCtrlMsgTitleFlagRed          = "RED\nFLAG"
	RaceCtrlMsgTitleFlagBW           = "BLACK\nWHITE"
	RaceCtrlMsgTitleFlagOther        = "FLAG"
	RaceCtrlMsgTitleFIA              = "FIA"
	RaceCtrlMsgTitleDefault          = "RACE\nCONTROL"
)

type RaceCtrlMsgCategory string

type RaceCtrlMsg struct {
	Category RaceCtrlMsgCategory
	Title    string
	Body     string
}
