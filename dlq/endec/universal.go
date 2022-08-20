package endec

type Universal struct{}

func (ed Universal) Encode(dst []byte, x interface{}) ([]byte, error) {
	return dst, nil
}

func (ed Universal) Decode(p []byte) (interface{}, error) {
	return nil, nil
}
