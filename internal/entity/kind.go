package entity

type Kind int

const (
	KindProgram Kind = iota
	KindICE
	KindVirus
)

func (k Kind) String() string {
	switch k {
	case KindProgram:
		return "Program"
	case KindICE:
		return "ICE"
	case KindVirus:
		return "Virus"
	default:
		return "Unknown"
	}
}
