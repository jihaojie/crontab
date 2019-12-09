package worker

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"project/crontab/common"
	"time"
)

//任务调度
type Scheduler struct {
	jobEventChan      chan *common.JobEvent              //etcd任务事件队列
	jobPlanTable      map[string]*common.JobSchedulePlan //任务调度计划表
	jobExecutingTable map[string]*common.JobExecuteInfo  //任务执行表
	jobResultChan     chan *common.JobExecuteResult      //任务结果队列
	config            clientv3.Config
	client            *clientv3.Client
}

var (
	G_Scheduler *Scheduler
)

//处理事件

func (scheduler *Scheduler) handleJobEvent(jobEvent *common.JobEvent) {
	var (
		jobSchedulePlan *common.JobSchedulePlan
		err             error
		jobExisted      bool
		jobExecuteInfo  *common.JobExecuteInfo
		jobExecuting    bool
	)
	switch jobEvent.EventType {
	case common.JOB_EVENT_SAVE:                                          //保存任务事件
		jobSchedulePlan, err = common.BuildJobSchedulePlan(jobEvent.Job) // 保存key
		if err != nil {
			return
		}

		scheduler.jobPlanTable[jobEvent.Job.Name] = jobSchedulePlan
	case common.JOB_EVENT_DELETE: // 删除任务事件

		//fmt.Println("-删除任务测试",scheduler.jobExecutingTable[jobEvent.Job.Name],jobEvent.Job.Name)

		jobSchedulePlan, jobExisted = scheduler.jobPlanTable[jobEvent.Job.Name]
		if jobExisted {
			delete(scheduler.jobPlanTable, jobEvent.Job.Name) // 删除key
		}
	case common.JOB_EVENT_KILL: //强杀任务事件
		//取消Command 执行
		//1.首先判断任务是否在执行中
		//fmt.Println(jobEvent.Job.Name, "触发强杀任务")
		jobExecuteInfo, jobExecuting = scheduler.jobExecutingTable[jobEvent.Job.Name]
		if jobExecuting { // 如果在执行
			jobExecuteInfo.CancelFuc() // 取消任务

		}

	}
}

//尝试执行任务
func (scheduler *Scheduler) TryStartJob(jobPlan *common.JobSchedulePlan) {
	//调度和执行是两件事情 需要考虑到调度的任务是否在执行
	//执行的任务可能运行很久,如果1秒的任务执行了60秒 那1分钟会调度60次 但是只能执行一次
	//去查询 jobExecutingTable 表里 如果有任务 说明在执行

	var (
		jobExecuteInfo *common.JobExecuteInfo
		ok             bool
	)

	//如果任务执行 跳过本次调度
	jobExecuteInfo, ok = scheduler.jobExecutingTable[jobPlan.Job.Name]

	if ok {
		fmt.Println("尚未执行完毕，跳过执行", jobPlan.Job.Name)
		return
	}

	//构建执行状态信息
	jobExecuteInfo = common.BuildJobExecuteInfo(jobPlan)

	//保存任务状态
	scheduler.jobExecutingTable[jobPlan.Job.Name] = jobExecuteInfo
	//fmt.Println(scheduler.jobExecutingTable,"-执行时列表")
	//执行任务

	//fmt.Println("执行任务", jobExecuteInfo.Job.Name, jobExecuteInfo.PlanTime, jobExecuteInfo.RealTime)
	G_executor.ExecuteJob(jobExecuteInfo)

}

//重新计算任务调度状态
//返回时间值 比如说 5秒后
func (scheduler *Scheduler) TrySchedule() (scheduleAfter time.Duration) {

	var (
		jobPlan  *common.JobSchedulePlan
		now      time.Time
		nearTime *time.Time
	)

	//如果任务列表为空 睡眠1秒 （返回一秒
	if len(scheduler.jobPlanTable) == 0 {
		scheduleAfter = 1 * time.Second
		return
	}

	now = time.Now()
	//fmt.Println("scheduler")
	for _, jobPlan = range scheduler.jobPlanTable {
		if jobPlan.NextTime.Before(now) || jobPlan.NextTime.Equal(now) {
			//fmt.Println("执行任务", jobPlan.Job.Name)
			scheduler.TryStartJob(jobPlan)
			jobPlan.NextTime = jobPlan.Expr.Next(now) // //无论怎样都需要计算并下次调度时间

		}

		//统计最近一个要过期的时间
		//== nil 就是从来没执行过 所以当前时间是最近的时间
		//等于遍历列表选出最接近的下次执行时间 进行赋值

		if nearTime == nil || jobPlan.NextTime.Before(*nearTime) {
			nearTime = &jobPlan.NextTime
		}

	}
	//下次调度时间间隔 (最近要执行的任务调度时间- 当前时间）
	scheduleAfter = (*nearTime).Sub(now)
	return
}

//处理任务结果
func (scheduler *Scheduler) handleJobResult(result *common.JobExecuteResult) {
	//删除执行状态
	//将返回结果存入数据库
	//生成执行日志

	var (
		jobLog *common.JobLog
		err    error
	)

	//锁抢占是正常的日志，所以需要忽略
	if result.Err != common.ERR_LOCK_ALREADY_REQUIRED {
		jobLog = &common.JobLog{
			JobName:      result.ExecuteInfo.Job.Name,
			Command:      result.ExecuteInfo.Job.Command,
			Output:       string(result.Output),
			PlanTime:     result.ExecuteInfo.PlanTime.UnixNano() / 1000 / 1000, // 纳秒->微秒->毫秒
			ScheduleTime: result.ExecuteInfo.RealTime.UnixNano() / 1000 / 1000,
			StartTime:    result.StartTime.UnixNano() / 1000 / 1000,
			EndTime:      result.EndTime.UnixNano() / 1000 / 1000,
		}
		if result.Err != nil {
			//默认是空指针，当有err字符的时候
			jobLog.Err = result.Err.Error()
		} else {
			jobLog.Err = ""
		}
	}
	//fmt.Println("推送日志-->")
	G_logSink.pushLog(jobLog) // 推送日志
	//TODO：查看 log 是否有错误 写到 etcd里 做一个通知 交给master 做报警
	if len(jobLog.Err) != 0 {
		err = scheduler.PushJobError(jobLog)
		if err != nil {
			return
		}
	}

	delete(scheduler.jobExecutingTable, result.ExecuteInfo.Job.Name)
	//fmt.Println("任务执行完成", result.ExecuteInfo.Job.Name, string(result.Output), result.Err)
}

//调度协程
func (scheduler *Scheduler) schedulerLoop() {
	//定时任务
	//fmt.Println("scheduler Loop")
	var (
		jobEvent      *common.JobEvent
		scheduleAfter time.Duration
		scheduleTimer *time.Timer
		jobResult     *common.JobExecuteResult
	)

	//初始化时间间隔，得到下次执行时间，第一次为1秒
	scheduleAfter = scheduler.TrySchedule()
	//延时定时器
	scheduleTimer = time.NewTimer(scheduleAfter)

	for {
		//select 会阻塞监听 任务变化事件 和timer定时器计时事件
		// 任务变化事件会修改任务列表
		// 两种事件都会触发任务的直接执行。
		select {

		case jobEvent = <-scheduler.jobEventChan: //监听任务变化事件
			//对内存中维护的列表任务做增删改查
			//fmt.Println(jobEvent.EventType, jobEvent.Job.Name)
			//scheduler.handleJobEvent(jobEvent) 下面全局为单例模式，
			G_Scheduler.handleJobEvent(jobEvent)

		case <-scheduleTimer.C: // 最近的任务到期了

		case jobResult = <-scheduler.jobResultChan: //监听任务执行结果
			scheduler.handleJobResult(jobResult)

		}
		scheduleAfter = scheduler.TrySchedule() //尝试执行任务，并返回最近的下次执行时间。

		//重置调度器 进行下一波
		scheduleTimer.Reset(scheduleAfter)

	}
}

//推送任务变化事件

func (scheduler *Scheduler) PushJobEvent(jobEvent *common.JobEvent) {

	scheduler.jobEventChan <- jobEvent
}

//初始化调度器
func InitScheduler() (err error) {
	//初始化了一个任务事件队列
	//任务执行计划表
	var (
		config clientv3.Config
		client *clientv3.Client
	)

	config = clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"}, //集群地址
		DialTimeout: 5000 * time.Millisecond,
	}
	//建立连接
	if client, err = clientv3.New(config); err != nil {
		return
	}

	G_Scheduler = &Scheduler{
		jobEventChan:      make(chan *common.JobEvent, 1000),
		jobPlanTable:      make(map[string]*common.JobSchedulePlan),
		jobExecutingTable: make(map[string]*common.JobExecuteInfo),
		jobResultChan:     make(chan *common.JobExecuteResult, 1000),
		config:            config,
		client:            client,
	}

	//启动调度协程
	go G_Scheduler.schedulerLoop()
	return

}

//回传任务执行结果

func (scheduler *Scheduler) PushJobResult(jobResult *common.JobExecuteResult) {

	scheduler.jobResultChan <- jobResult
}

func (scheduler Scheduler) PushJobError(log *common.JobLog) (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		errKey         string
		leaseID        clientv3.LeaseID
		errValue       string
	)
	leaseGrantResp, err = scheduler.client.Grant(context.TODO(), 10) // 10s 后过期
	if err != nil {
		return
	}

	errKey = common.JOB_EXECUTE_ERROR_DIR + log.JobName
	leaseID = leaseGrantResp.ID
	errValue = log.Err

	_, err = scheduler.client.KV.Put(context.TODO(), errKey, errValue, clientv3.WithLease(leaseID))

	return err
}
