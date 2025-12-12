package evm

import "bytes"

func (t *Builder) buildInstantiateCode(runtimeSize byte) (*bytes.Buffer, error) {
	dst := bytes.NewBuffer(make([]byte, 0))
	if _, err := dst.Write([]byte{OpPush1, runtimeSize}); err != nil {
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, 0x0c}); err != nil {
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, 0x00}); err != nil {
		return nil, err
	}
	if _, err := dst.Write([]byte{OpCodeCopy}); err != nil {
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, runtimeSize}); err != nil {
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, 0x00}); err != nil {
		return nil, err
	}
	if _, err := dst.Write([]byte{OpReturn}); err != nil {
		return nil, err
	}
	return dst, nil
}
