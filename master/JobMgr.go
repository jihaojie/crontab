package master

import (
	"go.etcd.io/etcd/clientv3"
	"time"
)

//任务管理器
type JobMgr struct {
	client *clientv3.Client // 客户端操作
	kv     clientv3.KV      //管理KV
	lease  clientv3.Lease   //续租
}

var (
	G_jobMgr *JobMgr
)

//初始化管理器 建立etcd的连接
func InitJobMgr() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
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

	//赋值单例
	G_jobMgr = &JobMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return

}



