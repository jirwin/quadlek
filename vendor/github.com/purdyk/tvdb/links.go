package tvdb

// Links for results

type Links struct {
	First int32 `json:"first,omitempty"`
	Next  int32 `json:"next,omitempty"`
	Prev  int32 `json:"prev,omitempty"`
	Last  int32 `json:"last,omitempty"`
}

func (l *Links) HasNext() bool {
	if l.Next == 0 {
		return false
	}

	return true
}
