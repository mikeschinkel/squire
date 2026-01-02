package gomtui

// Pane represents which pane currently has focus
type Pane int

const (
	LeftPane Pane = iota
	MiddlePane
	RightPane
)

func (p Pane) String() string {
	switch p {
	case LeftPane:
		return "Left"
	case MiddlePane:
		return "Middle"
	case RightPane:
		return "Right"
	default:
		return "Unknown"
	}
}
