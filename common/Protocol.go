package common

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorhill/cronexpr"
	"github.com/jmoiron/sqlx"
	"strings"
	"time"
)

//定时任务

type Job struct {
	Name     string `json:"name"`
	Command  string `json:"command"`
	CronExpr string `json:"cronExpr"`
}

//任务调度计划

type JobSchedulePlan struct {
	Job      *Job                 //要执行的任务信息
	Expr     *cronexpr.Expression //解析好的 cronexpr 表达式
	NextTime time.Time            //下次调度时间
}

//任务执行状态
type JobExecuteInfo struct {
	Job       *Job               // 任务信息
	PlanTime  time.Time          //理论上的调度时间
	RealTime  time.Time          //实际的调度时间
	CancelCtx context.Context    //任务command的上下文
	CancelFuc context.CancelFunc // 用于取消command执行的函数
}

//http接口应答

type Response struct {
	Errno int         `json:"errno"`
	Msg   string      `json:"msg"`
	Data  interface{} `json:"data"`
}

//变化事件
type JobEvent struct {
	EventType int //SAVE，DELETE
	Job       *Job
}

//任务执行日志
type JobLog struct {
	JobName      string `db:"jobName",json:"jobName"`           //任务名字
	Command      string `db:"command",json:"command"`           //脚本命令
	Err          string `db:"err",json:"err"`                   // 错误原因
	Output       string `db:"output",json:"output"`             //脚本输出
	PlanTime     int64  `db:"planTime",json:"planTime"`         // 计划调度开始时间
	ScheduleTime int64  `db:"scheduleTime",json:"scheduleTime"` // 真实调度时间
	StartTime    int64  `db:"startTime",json:"startTime"`       // 任务执行时间
	EndTime      int64  `db:"endTime",json:"endTime"`           // 任务执行结束时间
}

//任务执行结果
type JobExecuteResult struct {
	ExecuteInfo *JobExecuteInfo //执行的状态
	Output      []byte          // 脚本结果输出
	Err         error           //脚本错误原因
	StartTime   time.Time       //启动时间
	EndTime     time.Time       //结束时间
}

type LoginUser struct {
	Username string
	Password string
}

//应答方法
func BuildResponse(errno int, msg string, data interface{}) (resp Response, err error) {
	//定义一个response
	var (
		response Response
	)
	response.Errno = errno
	response.Msg = msg
	response.Data = data
	//序列化
	//resp, err = json.Marshal(response)
	resp = response
	return
}

//反序列化job
func UnpackJob(value []byte) (ret *Job, err error) {
	var (
		job *Job
	)
	job = &Job{}
	//如果报错了 就返回空指针和错误
	err = json.Unmarshal(value, &job)
	if err != nil {
		return
	}
	//没有报错就返回 序列化对象
	ret = job

	return
}

//从etcd的key中提取任务名
// /cron/jobs/job10 删除/cron/jobs/
func ExtractJobName(jobkey string) (string) {
	return strings.TrimPrefix(jobkey, JOB_SAVE_DIR)

}

//从etcd的key中提取任务名
// /cron/killer/job10 删除/cron/jobs/
func ExtractKillerName(killerKey string) (string) {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)

}

//提取Worker的IP
func ExtractWorkerIP(regKey string) (string) {
	return strings.TrimPrefix(regKey, JOB_WORKER_DIR)
}

//任务变化事件有两种,更新任务 删除任务

func BuildJobEvent(eventType int, job *Job) (jobEvent *JobEvent) {
	return &JobEvent{
		EventType: eventType,
		Job:       job,
	}
}

//构造任务执行计划
func BuildJobSchedulePlan(job *Job) (jobSchedulerPlan *JobSchedulePlan, err error) {
	var (
		expr *cronexpr.Expression
	)
	//解析job的cron表达式

	expr, err = cronexpr.Parse(job.CronExpr)
	if err != nil {
		return
	}

	//生成任务调度对象
	jobSchedulerPlan = &JobSchedulePlan{
		Job:      job,
		Expr:     expr,
		NextTime: expr.Next(time.Now()),
	}

	return
}

//构造执行状态信息

func BuildJobExecuteInfo(jobSchedulePlan *JobSchedulePlan) (jobExecuteInfo *JobExecuteInfo) {
	jobExecuteInfo = &JobExecuteInfo{
		Job:      jobSchedulePlan.Job,
		PlanTime: jobSchedulePlan.NextTime, //计算调度时间 （理论上的调度时间
		RealTime: time.Now(),               //真实的调度时间
	}
	jobExecuteInfo.CancelCtx, jobExecuteInfo.CancelFuc = context.WithCancel(context.TODO())
	return
}

func BuildDBInsertInfo(log *JobLog) {
	var (
		db  *sqlx.DB
		err error
	)

	tx := db.MustBegin()

	for i := 0; i <= 5; i++ {
		_, err = tx.NamedExec("INSERT INTO Log "+
			"(jobname,command,err,output,plantime,scheduletime,starttime,endtime) "+
			"VALUES (:jobName, :command, :err, :output, :planTime, :scheduleTime, :startTime,:endTime)", log)

	}

	err = tx.Commit()
	fmt.Println(err)

}

//发送短信
func SentMsg(phoneNumbers []int64, content string) {
	fmt.Println("发送短信-->", phoneNumbers, string(content))
}
