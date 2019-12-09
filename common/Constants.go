package common

const (
	//Etcd保存任务的目录
	JOB_SAVE_DIR = "/cron/jobs/"

	//Etcd任务强杀目录
	JOB_KILLER_DIR = "/cron/killer/"

	//任务锁目录
	JOB_LOCK_DIR = "/cron/lock/"

	//服务注册目录
	JOB_WORKER_DIR = "/cron/workers/"

	//任务执行失败目录
	JOB_EXECUTE_ERROR_DIR = "/cron/errors/"

	//保存任务事件
	JOB_EVENT_SAVE = 1

	//删除任务事件
	JOB_EVENT_DELETE = 2

	//杀死任务事件
	JOB_EVENT_KILL = 3



	//日志量达到多少条就同步一次数据库
	LOG_MAX_SAVE = 5

	//多久提交一次日志同步到数据库
	LOG_AUTO_COMMIT_TIMEOUT = 1

)
