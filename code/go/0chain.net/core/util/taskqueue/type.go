package taskqueue

type TaskType int

const (
	N2NMsg TaskType = iota // the value of the type is also the priority
	Common
	SCExec
	TypeNum
)

func (t TaskType) String() string {
	switch t {
	case SCExec:
		return "SCExec"
	case N2NMsg:
		return "N2NMsg"
	case Common:
		return "Common"
	default:
		panic("unknown task type")
	}
}
