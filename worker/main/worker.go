package main

import (
	"flag"
	"fmt"
	"project/crontab/worker"
	"time"
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
	/*  worker 节点

	 */
	var (
		err error
	)
	// 初始化命令参数
	//initArgs()

	//加载配置
	//if err = master.InitConfig(confFile); err != nil {
	//	goto ERR
	//}


	err = worker.InitLogSink()
	if err != nil {
		goto ERR
	}
	fmt.Println("日志同步模块已启动")
	err = worker.InitRegister()
	if err != nil {
		goto ERR
	}
	fmt.Println("配置中心已启动")
	err = worker.InitExecutor()
	if err != nil {
		goto ERR
	}

	//启动调度器
	//监听JobMgr的事件变化 维护内存调度列表 执行调度
	//初始化了一个任务调度列表 和 一个事件channel
	err = worker.InitScheduler()
	if err != nil {
		goto ERR
	}
	fmt.Println("调度器已启动")
	//启动任务管理器
	//初始化etcd的连接
	//第一次启动 把etcd中的任务都push给scheduler
	//后续一直监听etcd 中key 的变化 给scheduler发送任务事件
	err = worker.InitJobMgr()
	if err != nil {
		goto ERR
	}
	fmt.Println("任务管理器已启动")
	////启动任务监听
	//err = worker.G_jobMgr.WatchJobs()
	//if err != nil {
	//	goto ERR
	//}

	for {
		//永久性监听
		time.Sleep(1 * time.Second)
	}

ERR:
	fmt.Println(err)

}
