package worker

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"net"
	"project/crontab/common"
	"time"
)

//注册节点到etcd /cron/worker/IP
type Register struct {
	client  *clientv3.Client
	kv      clientv3.KV
	lease   clientv3.Lease
	localIP string // 本机IP
}

var (
	G_register *Register
)

//自动注册到 /cron/workers/IP， 并自动续租

func (register *Register) keepOnline() {
	var (
		regKey         string
		leaseGrantResp *clientv3.LeaseGrantResponse
		err            error
		keepAliveChan  <-chan *clientv3.LeaseKeepAliveResponse
		keepAliveResp  *clientv3.LeaseKeepAliveResponse
		cancelCtx      context.Context
		cancelFunx     context.CancelFunc
	)
	for {
		//注册路径
		regKey = common.JOB_WORKER_DIR + register.localIP

		cancelFunx = nil

		//创建租约
		leaseGrantResp, err = register.lease.Grant(context.TODO(), 10)
		if err != nil {
			goto RETRY
		}

		//自动续租
		keepAliveChan, err = register.lease.KeepAlive(context.TODO(), leaseGrantResp.ID)
		if err != nil {
			goto RETRY
		}

		cancelCtx, cancelFunx = context.WithCancel(context.TODO())

		//注册到etcd
		_, err = register.kv.Put(cancelCtx, regKey, "", clientv3.WithLease(leaseGrantResp.ID))
		if err != nil {
			goto RETRY
		}

		//处理续租应答
		for {
			select {
			case keepAliveResp = <-keepAliveChan:
				if keepAliveResp == nil { //续租失败
					goto RETRY
				}
			}
		}

	RETRY:
		time.Sleep(1 * time.Second)
		if cancelFunx != nil {
			cancelFunx()
		}
	}
}

func InitRegister() (err error) {
	var (
		config clientv3.Config
		client *clientv3.Client
		kv     clientv3.KV
		lease  clientv3.Lease
		//watcher clientv3.Watcher
		localip string
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

	localip, err = getLocalIP()
	if err != nil {
		return
	}

	kv = clientv3.NewKV(client)
	lease = clientv3.NewLease(client)
	//watcher = clientv3.NewWatcher(client)
	G_register = &Register{
		client:  client,
		kv:      kv,
		lease:   lease,
		localIP: localip,
	}

	go G_register.keepOnline() // 服务注册

	return

}

// 获取本机网卡IP
func getLocalIP() (ipv4 string, err error) {
	var (
		addrs   []net.Addr
		addr    net.Addr
		ipNet   *net.IPNet // IP地址
		isIpNet bool
	)
	// 获取所有网卡
	if addrs, err = net.InterfaceAddrs(); err != nil {
		return
	}
	// 取第一个非lo的网卡IP
	for _, addr = range addrs {
		// 这个网络地址是IP地址: ipv4, ipv6
		if ipNet, isIpNet = addr.(*net.IPNet); isIpNet && !ipNet.IP.IsLoopback() {
			// 跳过IPV6
			if ipNet.IP.To4() != nil {
				ipv4 = ipNet.IP.String() // 192.168.1.1
				return
			}
		}
	}

	err = common.ERR_NO_LOCAL_IP_FOUND
	return
}
