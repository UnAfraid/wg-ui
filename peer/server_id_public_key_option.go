package peer

type ServerIdPublicKeyOption struct {
	ServerId  string
	PublicKey string
}

func (option *ServerIdPublicKeyOption) Validate() error {
	if len(option.ServerId) == 0 {
		return ErrServerIdRequired
	}
	if len(option.PublicKey) == 0 {
		return ErrPublicKeyRequired
	}
	return nil
}
