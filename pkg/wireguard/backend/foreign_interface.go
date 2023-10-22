package backend

type ForeignInterface struct {
	Name      string
	Addresses []string
	Mtu       int
	State     string
}
