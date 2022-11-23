package config

type Server struct {
	User   map[string]*User `mapstructure:"user" json:"user" yaml:"user"`       //客户端信息
	Name   string           `mapstructure:"name" json:"name" yaml:"name"`       //tun 名称
	V4Addr string           `mapstructure:"v4Addr" json:"v4Addr" yaml:"v4Addr"` //V4Addr虚拟网卡的地址
	V6Addr string           `mapstructure:"v6Addr" json:"v6Addr" yaml:"v6Addr"` //tun虚拟网卡的地址
	Dns    string           `mapstructure:"dns" json:"dns" yaml:"dns"`          //tun DNS 地址
	Path   string           `mapstructure:"path" json:"path" yaml:"path"`       //http websocket 站点目录
	Port   string           `mapstructure:"port" json:"port" yaml:"port"`       //服务端监听的端口
}

type User struct {
	UUID string `mapstructure:"uuid" json:"uuid" yaml:"uuid"` //UUID 用来确定用户身份
	V4   string `mapstructure:"v4" json:"v4" yaml:"v4"`       //V4 客户端地址
	V6   string `mapstructure:"v6" json:"v6" yaml:"v6"`       //v6 客户端地址
	Key  string `mapstructure:"key" json:"key" yaml:"key"`    //二次加密数据的密钥
}
