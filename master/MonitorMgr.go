package master

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"project/crontab/common"
	"time"
)

//主要功能 master 监测报警key   worker 有错误任务信息 写到channel 里  同步到etcd
//设置一个key过期时间 这功能仅做报警使用

//监测etcd 的一个key

var (
	G_monitorMgr *MonitorMgr
)

type MonitorMgr struct {
	client  *clientv3.Client // 客户端操作
	kv      clientv3.KV      //管理KV
	lease   clientv3.Lease   //续租
	watcher clientv3.Watcher
}

func (monitorMgr *MonitorMgr) Monitor() {

	var (
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
	)

	go func() {
		//监听/cron/errors/目录的后续变化
		watchChan = monitorMgr.watcher.Watch(context.TODO(), common.JOB_EXECUTE_ERROR_DIR, clientv3.WithPrefix())

		//只要有新数据就会进行下一次for循环
		for watchResp = range watchChan {
			//遍历每一个事件
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //任务执行错误事件，监测到有任务执行错误
					//尝试发送短信
					//watchEvent.Kv.Value 为jobName  value 为output
					//发送短信模块还没搞完
					//TODO： 发送短信
					fmt.Println(string(watchEvent.Kv.Key))
					monitorMgr.TrySentMsg(string(watchEvent.Kv.Key), string(watchEvent.Kv.Value))

				case mvccpb.DELETE: // key标记过期，被自动删除

				}
			}
		}

	}()
}

func (monitorMgr MonitorMgr) TrySentMsg(jobName string, output string) {
	//发送短信

	var (
		//jobSentObj *common.JobErrorSentTable
		members []int64
	)

	members = make([]int64, 10)
	members = append(members, 123)
	common.SentMsg(members, jobName)
}

func InitMonitorMgr() (err error) {
	var (
		config  clientv3.Config
		client  *clientv3.Client
		kv      clientv3.KV
		lease   clientv3.Lease
		watcher clientv3.Watcher
	)
	//初始化配置
	config = clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"}, //集群地址
		DialTimeout: 5000 * time.Millisecond,
	}
	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}
	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	watcher = clientv3.NewWatcher(client)

	//赋值单例
	G_monitorMgr = &MonitorMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	go G_monitorMgr.Monitor() // 启动监测模块
	return
}
