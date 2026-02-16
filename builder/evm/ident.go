package evm

type IdentManager struct {
	offsetIdents map[string]byte
}

func (m *IdentManager) GetOffset(ident []byte) byte {
	return m.offsetIdents[string(ident)]
}

func (m *IdentManager) SetOffset(ident string, offset byte) {
	m.offsetIdents[ident] = offset
}

func (m *IdentManager) GetLength() uint {
	return uint(len(m.offsetIdents))
}

func NewIdentManager() *IdentManager {
	return &IdentManager{offsetIdents: make(map[string]byte)}
}
