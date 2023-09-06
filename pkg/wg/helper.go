package wg

func interfaceName(name string) []byte {
	b := make([]byte, 16)
	copy(b, name+"\x00")
	return b
}
