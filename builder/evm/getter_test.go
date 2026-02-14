package evm

import "testing"

func TestGetCalldataArgsOffset(t *testing.T) {
	cases := []struct {
		Name     string
		NthArg   uint64
		Expected byte
	}{
		{
			"sample_get_calldata_args_index_from_bytes_1",
			0,
			0x20, // 32 bytes
		},
		{
			"sample_get_calldata_args_index_from_bytes_2",
			1,
			0x40, // 64 bytes
		},
		{
			"sample_get_calldata_args_index_from_bytes_3",
			2,
			0x60, // 96 bytes (third slot)
		},
	}

	for _, c := range cases {
		got := GetCalldataArgsOffset(c.NthArg)
		t.Run(c.Name, func(t *testing.T) {
			expected := c.Expected
			if got != expected {
				t.Errorf("EVM get calldata args index from bytes: name: %v, got: %v, expected: %v", c.Name, got, expected)
			}
		})
	}
}
