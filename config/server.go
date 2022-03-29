package config

type Server struct {
	TunName string `mapstructure:"tunName" json:"tunName" yaml:"tunName"` //tun 名称
	TunAddr string `mapstructure:"tunAddr" json:"tunAddr" yaml:"tunAddr"` //tun虚拟网卡的地址
	TunGw   string `mapstructure:"tunGw" json:"tunGw" yaml:"tunGw"`       //tun虚拟网卡的网关
	TunMask string `mapstructure:"tunMask" json:"tunMask" yaml:"tunMask"` //tun子网掩码
	TunDns  string `mapstructure:"tunDns" json:"tunDns" yaml:"tunDns"`    //tun DNS 地址
	Type    string `mapstructure:"type" json:"type" yaml:"type"`          //选择隧道类型
	Path    string `mapstructure:"path" json:"path" yaml:"path"`          //http websocket 站点目录
	Port    int    `mapstructure:"port" json:"port" yaml:"port"`          //服务端监听的端口
	Key     string `mapstructure:"key" json:"key" yaml:"key"`             //加密密钥 uuid 也作为用户连接的密钥
	Encrypt bool   `mapstructure:"encrypt" json:"encrypt" yaml:"encrypt"` //是否开启https加密后的 第二次加密
}
