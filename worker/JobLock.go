package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"project/crontab/common"
)

//分布式锁 锁每个任务 (TXN事务抢key)

type Joblock struct {
	kv         clientv3.KV
	lease      clientv3.Lease
	jobName    string             //任务名
	cancelFunc context.CancelFunc //用于终止自动续租
	leaseid    clientv3.LeaseID   //租约ID
	isLocked   bool               //是否上锁成功
}

//初始化一把锁
func InitJobLock(jobName string, kv clientv3.KV, lease clientv3.Lease) (joblock *Joblock) {
	joblock = &Joblock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
	}
	return
}

//尝试上锁
func (jobLock *Joblock) TryLock() (err error) {

	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		cancelCtx      context.Context
		cancelFunc     context.CancelFunc
		leaseid        clientv3.LeaseID
		keepRespChan   <-chan *clientv3.LeaseKeepAliveResponse //只读channel
		txn            clientv3.Txn
		lockKey        string
		txnResp        *clientv3.TxnResponse
	)

	//1.创建租约
	leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5) // 创建一个5秒的租约
	if err != nil {
		return
	}
	//context 用于取消自动续租
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())

	//租约id
	leaseid = leaseGrantResp.ID

	//2.自动续租
	keepRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseid)
	if err != nil {
		goto FALL
	}
	//3.处理续租应答的协程

	go func() {

		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)

		for {
			select {
			case keepResp = <-keepRespChan: //自动续租的应答
				if keepResp == nil { //如果续租应答为nil 就不再消费续租应答
					goto END
				}

			}
		}
	END:
	}()

	//4.创建事务 txn
	txn = jobLock.kv.Txn(context.TODO())

	//锁路径
	lockKey = common.JOB_LOCK_DIR + jobLock.jobName

	//5.事务抢锁
	//如果创建版本等于0 （如果lockkey不存在) 就创建一个key  value 为"" 带一个租约
	//如果存在 就get一下这个key
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseid))).
		Else(clientv3.OpGet(lockKey))

	txnResp, err = txn.Commit()
	if err != nil {
		goto FALL
	}

	//6.成功返回 失败释放租约
	if !txnResp.Succeeded {
		//如果没有成功 说明锁被占用
		err = common.ERR_LOCK_ALREADY_REQUIRED
		goto FALL
	}
	//抢锁成功
	jobLock.leaseid = leaseid //解锁时候用
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true
	return

FALL:
	//保证在任何错误情况下 都要释放租约
	cancelFunc()                                  // 取消自动续租
	jobLock.lease.Revoke(context.TODO(), leaseid) //释放租约
	return

}

func (jobLock *Joblock) Unlock() {
	if jobLock.isLocked {
		//如果已经上锁
		jobLock.cancelFunc()                                  // 取消我们程序自动续租的协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseid) // 释放租约 key会自动删除
	}
}
