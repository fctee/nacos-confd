package backends

import (
	util "github.com/Risingtao/nacos-confd/util"
)

type Config struct {
	Backend      string     `toml:"backend"`
	BackendNodes util.Nodes `toml:"nodes"`
	Password     string     `toml:"password"`
	Endpoint     string     `toml:"endpoint"`
	Namespace    string     `toml:"namespace"`
	AccessKey    string     `toml:"accessKey"`
	SecretKey    string     `toml:"secretKey"`
	OpenKMS      bool       `toml:"openKMS"`
	RegionId     string     `toml:"regionId"`
}
