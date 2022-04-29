package config

type Server struct {
	Name   string `mapstructure:"name" json:"name" yaml:"name"`       //tun 名称
	V4Addr string `mapstructure:"v4Addr" json:"v4Addr" yaml:"v4Addr"` //V4Addr虚拟网卡的地址
	V4Gw   string `mapstructure:"v4Gw" json:"v4Gw" yaml:"v4Gw"`       //V4Gw虚拟网卡的网关
	V4Mask string `mapstructure:"v4Mask" json:"v4Mask" yaml:"v4Mask"` //V4Mask子网掩码
	Dns    string `mapstructure:"dns" json:"dns" yaml:"dns"`          //tun DNS 地址
	Type   string `mapstructure:"type" json:"type" yaml:"type"`       //选择隧道类型
	Path   string `mapstructure:"path" json:"path" yaml:"path"`       //http websocket 站点目录
	Port   string `mapstructure:"port" json:"port" yaml:"port"`       //服务端监听的端口
	Key    string `mapstructure:"key" json:"key" yaml:"key"`          //加密密钥 uuid 也作为用户连接的密钥

	CertFile string `mapstructure:"certFile" json:"certFile" yaml:"certFile"` //ssl 证书
	KeyFile  string `mapstructure:"keyFile" json:"keyFile" yaml:"keyFile"`    //ssl证书key
}
