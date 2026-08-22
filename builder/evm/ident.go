package evm

// An IdentManager gives each name of one scope a place in that scope's frame.
//
// It belongs to a scope and not to a contract. It used to belong to the contract, so two
// scopes binding a name each were given the same place — and two activations of one scope
// were as well, which is what kept a scope from calling itself.
type IdentManager struct {
	offsetIdents map[string]int
	base         int
}

// Base is how far into the frame this scope's names begin: past the places the values applied
// to it are kept.
func (m *IdentManager) Base() int {
	return m.base
}

func (m *IdentManager) GetOffset(ident []byte) int {
	return m.offsetIdents[string(ident)]
}

func (m *IdentManager) SetOffset(ident string, offset int) {
	m.offsetIdents[ident] = offset
}

func (m *IdentManager) GetLength() uint {
	return uint(len(m.offsetIdents))
}

func NewIdentManager() *IdentManager {
	return NewIdentManagerAt(0)
}

// NewIdentManagerAt makes one whose names begin at an offset of the frame.
func NewIdentManagerAt(base int) *IdentManager {
	return &IdentManager{offsetIdents: make(map[string]int), base: base}
}
