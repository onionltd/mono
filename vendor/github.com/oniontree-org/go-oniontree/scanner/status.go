package scanner

type Status uint8

func (s Status) String() string {
	switch s {
	case StatusOnline:
		return "online"
	case StatusOffline:
		return "offline"
	}
	return ""
}

const (
	StatusOnline  Status = 1
	StatusOffline Status = 0
)
