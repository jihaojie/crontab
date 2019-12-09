package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"project/crontab/common"
	"time"
)

//任务管理器
type JobMgr struct {
	client  *clientv3.Client // 客户端操作
	kv      clientv3.KV      //管理KV
	lease   clientv3.Lease   //续租
	watcher clientv3.Watcher
}

var (
	G_jobMgr *JobMgr
)

func (jobMgr *JobMgr) WatchJobs() (err error) {
	var (
		getResp            *clientv3.GetResponse
		kvpair             *mvccpb.KeyValue
		job                *common.Job
		watchStartRevision int64
		watchChan          clientv3.WatchChan
		watchResp          clientv3.WatchResponse
		watchEvent         *clientv3.Event
		jobName            string
		jobEvent           *common.JobEvent
	)

	//1.get 一下 //cron/jobs/目录下的所有任务，并且获知当前集群的revision
	//2.从该revision向后监听变化事件

	getResp, err = jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix())

	if err != nil {
		return
	}

	for _, kvpair = range getResp.Kvs {

		//反序列化json 得到job

		job, err = common.UnpackJob(kvpair.Value)

		if err == nil {
			//把任务同步给scheduler(调度协程)
			jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)

			G_Scheduler.PushJobEvent(jobEvent)
		}

	}

	//2.从该revision向后监听变化事件
	go func() { //监听协程
		//从get 时刻的后续版本开始监听变化
		watchStartRevision = getResp.Header.Revision + 1
		//监听/cron/jobs/目录的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithRev(watchStartRevision), clientv3.WithPrefix())

		//只要有新数据就会进行下一次for循环
		for watchResp = range watchChan {
			//遍历每一个事件
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT: //任务保存事件
					//反序列化job 推送更新事件给scheduler
					job, err = common.UnpackJob(watchEvent.Kv.Value)
					if err != nil {
						//如果反序列化失败，忽略这次任务 继续监听
						continue
					}
					//构建一个Event事件
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_SAVE, job)
					G_Scheduler.PushJobEvent(jobEvent)

				case mvccpb.DELETE: // 任务删除事件
					//推送一个 删除事件给scheduler
					//Delete /cron/jobs/job10
					jobName = common.ExtractJobName(string(watchEvent.Kv.Key)) //得到job10
					job = &common.Job{Name: jobName}                           // 删除事件只需要知道jobname
					//构造一个删除Event
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_DELETE, job)
					G_Scheduler.PushJobEvent(jobEvent)
				}
			}
		}
	}()

	return
}

//初始化管理器 建立etcd的连接
func InitJobMgr() (err error) {
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
	G_jobMgr = &JobMgr{
		client:  client,
		kv:      kv,
		lease:   lease,
		watcher: watcher,
	}

	//启动任务监听
	G_jobMgr.WatchJobs()

	//启动killer监听
	G_jobMgr.watchKiller()

	return

}

//创建任务执行锁

func (jobMgr *JobMgr) CreateJobLock(jobName string) (jobLock *Joblock) {
	//返回一把锁
	jobLock = InitJobLock(jobName, jobMgr.kv, jobMgr.lease)
	return
}

//监听强杀任务通知

func (jobMgr *JobMgr) watchKiller() {
	//监听/cron/killer 目录
	var (
		watchChan  clientv3.WatchChan
		watchResp  clientv3.WatchResponse
		watchEvent *clientv3.Event
		jobEvent   *common.JobEvent
		jobName    string
		job        *common.Job
	)

	go func() { //监听协程
		//监听/cron/killer/目录的后续变化
		watchChan = jobMgr.watcher.Watch(context.TODO(), common.JOB_KILLER_DIR, clientv3.WithPrefix())

		//只要有新数据就会进行下一次for循环
		for watchResp = range watchChan {
			//遍历每一个事件
			for _, watchEvent = range watchResp.Events {
				switch watchEvent.Type {
				case mvccpb.PUT:                                                  //杀死任务事件
					jobName = common.ExtractKillerName(string(watchEvent.Kv.Key)) // /cron/killer/job10
					job = &common.Job{Name: jobName}
					jobEvent = common.BuildJobEvent(common.JOB_EVENT_KILL, job)

					//推给scheduler
					G_Scheduler.PushJobEvent(jobEvent)
				case mvccpb.DELETE: // killer标记过期，被自动删除

				}
			}
		}
	}()

}
