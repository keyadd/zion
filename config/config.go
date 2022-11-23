package config

type Config struct {
	Client Client `mapstructure:"client" json:"client" yaml:"client"` //客户端
	Server Server `mapstructure:"server" json:"server" yaml:"server"` //服务端
}
