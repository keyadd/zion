package config

type Config struct {
	Client Client `mapstructure:"client" json:"client" yaml:"client"`
	Server Server `mapstructure:"server" json:"server" yaml:"server"`
}
