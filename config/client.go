package config

type Client struct {
	Name    string `mapstructure:"name" json:"name" yaml:"name"`          //tun 名称
	V4Addr  string `mapstructure:"v4Addr" json:"v4Addr" yaml:"v4Addr"`    //tun虚拟网卡的地址
	V4Gw    string `mapstructure:"v4Gw" json:"v4Gw" yaml:"v4Gw"`          //tun虚拟网卡的网关
	V4Mask  string `mapstructure:"v4Mask" json:"v4Mask" yaml:"v4Mask"`    //tun子网掩码
	Dns     string `mapstructure:"dns" json:"dns" yaml:"dns"`             //tun DNS 地址
	Type    string `mapstructure:"type" json:"type" yaml:"type"`          //选择隧道类型
	Path    string `mapstructure:"path" json:"path" yaml:"path"`          //http websocket 站点目录
	Addr    string `mapstructure:"addr" json:"addr" yaml:"addr"`          //客户端连接的服务器地址
	Key     string `mapstructure:"key" json:"key" yaml:"key"`             //加密密钥 uuid 也作为用户连接的密钥
	TLS     bool   `mapstructure:"tls" json:"tls" yaml:"tls"`             //是否开启 https 加密
	Encrypt bool   `mapstructure:"encrypt" json:"encrypt" yaml:"encrypt"` //是否开启https加密后的 第二次加密
}
