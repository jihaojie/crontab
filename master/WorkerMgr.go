package master

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"project/crontab/common"
	"time"
)

// /cron/workers/ 所有的k,v
type WorkerMgr struct {
	client *clientv3.Client
	kv     clientv3.KV
	lease  clientv3.Lease
}

var (
	G_workerMgr *WorkerMgr
)

//获取在线worker列表
func (WorkerMgr *WorkerMgr) ListWorks() (workerArr []string, err error) {
	var (
		getResp  *clientv3.GetResponse
		kv       *mvccpb.KeyValue
		workerIP string
	)
	//初始化数组
	workerArr = make([]string, 0)
	//获取目录下所有KV

	getResp, err = WorkerMgr.kv.Get(context.TODO(), common.JOB_WORKER_DIR, clientv3.WithPrefix())
	if err != nil {
		return
	}
	//解析每个节点的IP
	for _, kv = range getResp.Kvs {
		//kv.key : /cron/workers/192.168.1.1

		workerIP = common.ExtractWorkerIP(string(kv.Key))
		workerArr = append(workerArr, workerIP)
	}

	return
}

func InitWorkerMgr() (err error) {

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

	G_workerMgr = &WorkerMgr{
		client: client,
		kv:     kv,
		lease:  lease,
	}

	return
}
