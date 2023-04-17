package config

type Initial struct {
	Email    string `default:"admin@example.com"`
	Password string `default:"random"`
}
