package master

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	ApiPort         int `json:"api_port"`
	ApiReadTimeout  int `json:"api_read_timeout"`
	ApiWriteTimeout int `json:"api_write_timeout"`
}

var (
	//单例
	G_config *Config
)

func InitConfig(filename string) (err error) {
	//1.把配置文件读进来
	var (
		content []byte
		conf    Config
	)

	content, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	//2.json 反序列化
	if err = json.Unmarshal(content, &conf); err != nil {
		return
	}

	//3.赋值单例
	G_config = &conf
	return
}
