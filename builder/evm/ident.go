package evm

type IdentManager struct {
	offsetIdents map[string]int
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
	return &IdentManager{offsetIdents: make(map[string]int)}
}
