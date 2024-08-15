package environ

type Claim interface {
	Bytes() []byte
}

type ca []byte

func (c ca) Bytes() []byte {
	return c
}

func TransportClaim(bs []byte) *ca {
	return (*ca)(&bs)
}
