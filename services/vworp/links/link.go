package links

type Link struct {
	URL       string
	ServiceID string
	Status    Status
}

func (l Link) String() string {
	return l.URL
}
