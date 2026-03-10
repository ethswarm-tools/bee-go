package swarm

// Reference represents a Swarm reference (hash).
type Reference struct {
	Value string
}

func (r Reference) String() string {
	return r.Value
}
