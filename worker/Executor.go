package worker

import (
	"math/rand"
	"os/exec"
	"project/crontab/common"
	"time"
)

//任务执行器

type Executor struct {
}

var (
	G_executor *Executor
)

func (executor *Executor) ExecuteJob(info *common.JobExecuteInfo) {

	//启动协程调度
	go func() {
		var (
			cmd     *exec.Cmd
			err     error
			output  []byte
			result  *common.JobExecuteResult
			jobLock *Joblock
		)

		//任务结果
		result = &common.JobExecuteResult{
			ExecuteInfo: info,
			Output:      make([]byte, 0),
		}

		//首先获取分布式锁
		//获取到了继续执行，不然直接返回
		jobLock = G_jobMgr.CreateJobLock(info.Job.Name) //根据key 获取一把锁

		//记录任务开始时间
		result.StartTime = time.Now()

		//抢锁时，随机睡眠0~100毫秒，来进行平均调度
		time.Sleep(time.Duration(rand.Intn(100)) * time.Microsecond)
		err = jobLock.TryLock()

		defer jobLock.Unlock() //释放锁

		if err != nil { //上锁失败
			result.Err = err
			result.EndTime = time.Now()
		} else {
			//上锁成功后，重置任务启动时间
			result.StartTime = time.Now()

			//任务执行完成后，把执行的结果返回给scheduler
			//scheduler 会从executingTable中删除掉执行记录
			cmd = exec.CommandContext(info.CancelCtx, "/bin/bash", "-c", info.Job.Command)
			output, err = cmd.CombinedOutput()

			//记录任务结束时间
			result.EndTime = time.Now()
			result.Output = output
			result.Err = err
		}

		//任务执行完毕后，把执行的结果返回给Scheduler，scheduler会从executingTable中删除执行记录
		G_Scheduler.PushJobResult(result)
	}()

}
//初始化执行器

func InitExecutor() (err error) {
	G_executor = &Executor{

	}
	return
}
