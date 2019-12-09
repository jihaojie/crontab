package main

import (
	"flag"
	"fmt"
	"project/crontab/master"
)

//解析命令行参数
var (
	confFile string // 配置文件路径
)

func initArgs() {
	//master -config ./master.json

	flag.StringVar(&confFile, "config", "./master.json", "传入master.json")
	flag.Parse()

}

func main() {
	var (
		err error
	)
	// 初始化命令参数
	//initArgs()

	//加载配置
	//if err = master.InitConfig(confFile); err != nil {
	//	goto ERR
	//}

	err = master.InitMonitorMgr()
	if err != nil {
		goto ERR // 报警模块
	}

	err = master.InitDBMgr() // 连接数据库模块
	if err != nil {
		goto ERR
	}

	//初始化服务发现模块
	err = master.InitWorkerMgr() //链接etcd
	if err != nil {
		goto ERR
	}

	//启动任务管理器
	if err = master.InitJobMgr(); err != nil {
		fmt.Println(err)
		goto ERR
	}

	//启动Api HTTP服务
	master.StartApiServer()

	return
ERR:
	fmt.Println(err)

	//正常退出
}
