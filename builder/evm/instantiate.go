package evm

import "bytes"

const INSTANTIATE_CODE_SIZE = 12

func (t *Builder) buildInstantiateCode(runtimeSize byte) (*bytes.Buffer, error) {
	dst := bytes.NewBuffer(make([]byte, 0))
	if _, err := dst.Write([]byte{OpPush1, runtimeSize}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, 0x0c}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, 0x00}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := dst.Write([]byte{OpCodeCopy}); err != nil { // 1 byte
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, runtimeSize}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := dst.Write([]byte{OpPush1, 0x00}); err != nil { // 2 bytes
		return nil, err
	}
	if _, err := dst.Write([]byte{OpReturn}); err != nil { // 1 byte
		return nil, err
	}
	return dst, nil
}
