package master

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"net/http"
	"project/crontab/common"
)

//任务的HTTP接口
type ApiServer struct {
	httpServer *http.Server
}

var (
	//单例对象
	G_apiServer *ApiServer
)

func checkLogin() gin.HandlerFunc {

	return func(c *gin.Context) {
		_, err := c.Cookie("Crontab")
		if err != nil {
			c.Redirect(302, "/account/login/")
		}
		c.Next()
	}
}

//gin 框架初始化服务
func StartApiServer() {

	router := gin.Default()
	router.Static("/static", "project/crontab/master/main/static")
	router.LoadHTMLGlob("project/crontab/master/main/webroot/*.html")
	//router.Use(checkLogin())
	//router.GET("/login/", controllers.GetLogin)

	router.POST("/account/login", handleLogin)
	router.GET("/account/login", webLoginIndex)
	router.GET("/index", checkLogin(), webIndex)
	router.POST("/job/save", handleJobSave)
	router.POST("/job/delete", handleJobDelete)
	router.GET("/job/list", handleJobList)
	router.POST("/job/kill", handleJobKill)
	router.GET("/worker/list", handleWorkList)
	router.POST("/job/log", handleJobLog)

	router.Run("127.0.0.1:8080")

}

func handleLogin(c *gin.Context) {
	var (
		username string
		password string
		isExist  bool
		err      error
		bytes    common.Response
	)

	username = c.PostForm("username")
	password = c.PostForm("password")
	isExist, err = G_dbMgr.IsAccountExists(username, password)

	if err != nil {
		if !isExist {
			err = common.ERR_ACCOUNT_NOT_FOUND
		}
		goto ERR

	}

	c.SetCookie("Crontab", "true", 3600, "/", "127.0.0.1", false, true)
	c.Redirect(302, "/index/")

	//bytes, err = common.BuildResponse(0, "success", isExist)
	//if err == nil {
	//	c.JSON(200, bytes)
	//}

	return

ERR:
	//返回错误应答
	bytes, err = common.BuildResponse(-1, err.Error(), nil)
	if err == nil {
		c.JSON(200, bytes)
	}

}

//登录页面
func webLoginIndex(c *gin.Context) {
	c.HTML(200, "login_v1.html", gin.H{})
}

//index页面
func webIndex(c *gin.Context) {

	c.HTML(200, "crontab-index.html", gin.H{})

}

func handleJobLog(c *gin.Context) {
	var (
		name  string
		log   *common.JobLog
		err   error
		bytes common.Response
	)

	name = c.PostForm("name")

	log, err = G_dbMgr.GetLog(name)
	if err != nil {
		goto ERR
	}

	bytes, err = common.BuildResponse(0, "success", log)
	if err == nil {
		c.JSON(200, bytes)

	}
	return

ERR:
	//返回错误应答
	bytes, err = common.BuildResponse(-1, err.Error(), nil)
	if err == nil {
		c.JSON(200, bytes)
	}
}

//获取健康worker节点列表
func handleWorkList(c *gin.Context) {
	var (
		workerArr []string
		err       error
		bytes     common.Response
	)
	workerArr, err = G_workerMgr.ListWorks()
	if err != nil {
		goto ERR
	}

	bytes, err = common.BuildResponse(0, "success", workerArr)
	if err == nil {
		c.JSON(200, bytes)

	}
	return

ERR:
	//返回错误应答
	bytes, err = common.BuildResponse(-1, err.Error(), nil)
	if err == nil {
		c.JSON(200, bytes)
	}
}

//强制杀死某个任务
func handleJobKill(c *gin.Context) {
	var bytes common.Response

	name := c.PostForm("name")
	err := G_jobMgr.KillJob(name)
	if err != nil {
		goto ERR
	}
	//返回正常应答
	bytes, err = common.BuildResponse(0, "success", nil)
	if err == nil {
		c.JSON(200, bytes)

	}

	return
ERR:
	//返回错误应答
	bytes, err = common.BuildResponse(-1, err.Error(), nil)
	if err == nil {
		c.JSON(200, bytes)
	}

}

//获取所有任务的接口
func handleJobList(c *gin.Context) {
	var (
		jobList []*common.Job
		err     error
		bytes   common.Response
	)

	jobList, err = G_jobMgr.ListJobs()
	if err != nil {
		goto ERR
	}

	//返回正常应答
	bytes, err = common.BuildResponse(0, "success", jobList)
	if err == nil {
		c.JSON(200, bytes)

	}

	return
ERR:
	//返回错误应答
	bytes, err = common.BuildResponse(-1, err.Error(), nil)
	if err == nil {
		c.JSON(200, bytes)
	}

}

//删除任务接口 POST /job/delete job1
func handleJobDelete(c *gin.Context) {

	var (
		bytes common.Response
	)
	name := c.PostForm("name")
	//fmt.Println(name,"--->")
	//删除任务
	oldJob, err := G_jobMgr.DeleteJob(name)

	if err != nil {
		fmt.Println(err)
		goto ERR
	}

	//返回正常应答
	bytes, err = common.BuildResponse(0, "success", oldJob)
	if err == nil {
		c.JSON(200, bytes)

	}

	return
ERR:
	//返回错误应答
	bytes, err = common.BuildResponse(-1, err.Error(), nil)
	if err == nil {
		c.JSON(200, bytes)
	}
}

//保存任务接口
func handleJobSave(c *gin.Context) {
	//任务保存到etcd中
	//解析Post表单
	var (
		err error
		job common.Job
		//bytes []byte
		oldJob *common.Job
		bytes  common.Response
	)

	//2 取表单中的job字段
	postJob := c.PostForm("job")
	//fmt.Println(postJob,"-->")
	//3.反序列化job
	if err = json.Unmarshal([]byte(postJob), &job); err != nil {
		fmt.Println(err)
		goto ERR
	}

	//4.保存到etcd
	oldJob, err = G_jobMgr.SaveJob(&job)
	if err != nil {
		fmt.Println(err)
		goto ERR
	}

	//5.返回正常应答
	bytes, err = common.BuildResponse(0, "success", oldJob)
	if err == nil {
		c.JSON(200, bytes)
	}
	return

ERR:
	//返回错误应答
	bytes, err = common.BuildResponse(-1, err.Error(), nil)
	if err == nil {
		c.JSON(200, bytes)
	}

}

//保存任务
func (jobMgr *JobMgr) SaveJob(job *common.Job) (oldJob *common.Job, err error) {

	//把任务保存到 /cron/jobs/任务名 -> json
	var (
		jobkey    string
		jobValue  []byte
		putResp   *clientv3.PutResponse
		oldJobObj common.Job
	)

	jobkey = common.JOB_SAVE_DIR + job.Name
	jobValue, err = json.Marshal(job)
	if err != nil {
		fmt.Println(err)
		return
	}
	//保存到etcd  put一个key 获取之前的key
	putResp, err = jobMgr.kv.Put(context.TODO(), jobkey, string(jobValue), clientv3.WithPrevKV())
	if err != nil {
		fmt.Println(err)
		return
	}
	//如果是更新 返回旧值。
	if putResp.PrevKv != nil {
		//对旧值做一个反序列化
		err = json.Unmarshal(putResp.PrevKv.Value, &oldJobObj)
		if err != nil {
			err = nil //旧值不影响新值的赋值
			return
		}

		oldJob = &oldJobObj
	}
	return
}

//删除任务
func (jobMgr *JobMgr) DeleteJob(job string) (oldJob *common.Job, err error) {

	var (
		jobkey    string
		delResp   *clientv3.DeleteResponse
		oldJobObj common.Job
	)

	//etcd中任务名字
	jobkey = common.JOB_SAVE_DIR + job
	//fmt.Println(jobkey, "delete key ")
	//删除key
	delResp, err = jobMgr.kv.Delete(context.TODO(), jobkey, clientv3.WithPrevKV())
	if err != nil {
		return
	}

	//fmt.Println(len(delResp.PrevKvs), "len")
	//返回被删除的任务信息
	if len(delResp.PrevKvs) != 0 {

		//如果不等于0 说明没有旧值
		//旧值解析 返回
		err = json.Unmarshal(delResp.PrevKvs[0].Value, &oldJobObj)
		if err != nil {
			//因为不太关心旧值 所以错误为nil
			//fmt.Println(err, "err")
			err = nil
			return
		}

		oldJob = &oldJobObj
		//fmt.Println("oldjob", oldJob.Name)
	}

	return
}

//列举任务
func (jobMgr JobMgr) ListJobs() (jobList []*common.Job, err error) {
	var (
		//dirKey  string
		getResp *clientv3.GetResponse
		kvPair  *mvccpb.KeyValue
		//jobList []*common.Job
		job *common.Job
	)
	//获取目录下所有的key  prefix 匹配
	//dirKey = common.JOB_SAVE_DIR

	//获取目录下所有任务信息
	getResp, err = jobMgr.kv.Get(context.TODO(), common.JOB_SAVE_DIR, clientv3.WithPrefix())
	if err != nil {
		return
	}

	jobList = make([]*common.Job, 0)
	//len(jobList) == 0

	//遍历所有任务 进行反序列化
	for _, kvPair = range getResp.Kvs {
		job = &common.Job{}
		err = json.Unmarshal(kvPair.Value, job)
		if err != nil {
			//忽略错误的原因是 数据结构一般都是确定的
			err = nil
			continue
		}
		//把反序列化的数据加到数组里
		jobList = append(jobList, job)
	}

	return

}

//杀死任务
func (jobMgr JobMgr) KillJob(name string) (err error) {
	//更新一下key= /cron/killer/任务名
	//所有worker监听任务变化，杀死任务
	//这里就是做了一个 put timeout = 1 的一个key
	var (
		killerKey string
		leaseResp *clientv3.LeaseGrantResponse
		leaseID   clientv3.LeaseID
	)
	killerKey = common.JOB_KILLER_DIR + name

	//让worker 监听一次put操作 设定一个过期时间
	//因为只是作为一个通知操作
	leaseResp, err = jobMgr.lease.Grant(context.TODO(), 2)
	if err != nil {
		return
	}
	//租约ID
	leaseID = leaseResp.ID

	//设置killer标记
	_, err = jobMgr.kv.Put(context.TODO(), killerKey, "", clientv3.WithLease(leaseID))
	if err != nil {
		return
	}
	return

}

func (dbmgr DBMgr) IsAccountExists(username string, password string) (IsLogin bool, err error) {

	var (
		sqlStr string
		user   common.LoginUser
	)

	sqlStr = "select username,password from LoginInfo where username = ? and password = ?"

	err = G_dbMgr.client.Get(&user, sqlStr, username, password)
	if err != nil {
		return
	}
	IsLogin = true // 获取到数据
	return
}

func (dbmgr DBMgr) GetLog(jobname string) (resultLog *common.JobLog, err error) {
	var (
		sqlStr string
		log    common.JobLog
	)
	//fmt.Println(jobname)
	sqlStr = "select jobName,command,err,output,startTime,endTime from Log where jobname = ? order by id desc limit 1"
	err = G_dbMgr.client.Get(&log, sqlStr, jobname)
	if err != nil {
		fmt.Println(err)
		return
	}
	resultLog = &log
	return
}
