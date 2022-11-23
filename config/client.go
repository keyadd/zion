package config

type Client struct {
	Name    string `mapstructure:"name" json:"name" yaml:"name"`          //tun 名称
	UUID    string `mapstructure:"uuid" json:"uuid" yaml:"uuid"`          //tun 名称
	V4Addr  string `mapstructure:"v4Addr" json:"v4Addr" yaml:"v4Addr"`    //tun虚拟网卡的地址
	V6Addr  string `mapstructure:"v6Addr" json:"v6Addr" yaml:"v6Addr"`    //tun虚拟网卡的地址
	Dns     string `mapstructure:"dns" json:"dns" yaml:"dns"`             //tun DNS 地址
	Path    string `mapstructure:"path" json:"path" yaml:"path"`          //http websocket 站点目录
	Addr    string `mapstructure:"addr" json:"addr" yaml:"addr"`          //客户端连接的服务器地址
	Key     string `mapstructure:"key" json:"key" yaml:"key"`             //二次加密数据的密钥
	TLS     bool   `mapstructure:"tls" json:"tls" yaml:"tls"`             //是否开启 https 加密
	Encrypt bool   `mapstructure:"encrypt" json:"encrypt" yaml:"encrypt"` //是否开启https加密后的 第二次加密
}
