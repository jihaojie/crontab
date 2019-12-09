package worker

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"project/crontab/common"
	"time"
)

//存储日志

//思路  1，初始化链接操作， 2. 监听channel  3.优化写入，3秒同步或者满足100条。

type LogSink struct {
	logChan chan *common.JobLog // 接收日志的channel
	client  *sqlx.DB            // 连接数据库的客户端
}

var (
	G_logSink *LogSink
)

func (logSink LogSink) writeLoop() {
	var (
		jobLog      *common.JobLog
		logArr      []*common.JobLog
		commitTimer *time.Timer
	)

	commitTimer = time.NewTimer(common.LOG_AUTO_COMMIT_TIMEOUT * time.Second)

	//写入逻辑，每隔一秒写入一次 减少IO操作
	for {
		select {
		case jobLog = <-G_logSink.logChan: // 监听Log队列里的数据
			logArr = append(logArr, jobLog)
			//if len(logArr) > common.LOG_MAX_SAVE { // 如果日志条数达到100条 才写入数据库 减少IO操作
			//	//fmt.Println(logArr,"--->")
			//	G_logSink.saveLog(logArr)

		case <-commitTimer.C: // timer到期 ，写入数据库
			G_logSink.saveLog(logArr)
			logArr = nil //写入完毕清空数组
		}

		commitTimer.Reset(common.LOG_AUTO_COMMIT_TIMEOUT * time.Second) // 重置调度时间
	}
}

//推送日志到channel
func (logsink LogSink) pushLog(log *common.JobLog) {

	G_logSink.logChan <- log
}

//保存日志
func (logsink LogSink) saveLog(logArr []*common.JobLog) {

	var (
		log *common.JobLog
		tx  *sqlx.Tx
		err error
	)

	tx = G_logSink.client.MustBegin() //事务开始

	for _, log = range logArr {
		_, err = tx.NamedExec("INSERT INTO Log "+
			"(jobname,command,err,output,plantime,scheduletime,starttime,endtime) "+
			"VALUES (:jobName, :command, :err, :output, :planTime, :scheduleTime, :startTime,:endTime)", log)

	}

	err = tx.Commit() //提交事务
	if err != nil {
		fmt.Println(err) //只打印事务错误
	}

}

func InitLogSink() (err error) {
	var (
		db *sqlx.DB
	)
	dsn := "jihaojie:123456@tcp(127.0.0.1:3306)/crontab"
	// 也可以使用MustConnect连接不成功就panic
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		fmt.Printf("connect DB failed, err:%v\n", err)
		return
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	G_logSink = &LogSink{
		logChan: make(chan *common.JobLog, 1000), // 
		client:  db,
	}

	go G_logSink.writeLoop() // 监听channel同步日志

	return

}
