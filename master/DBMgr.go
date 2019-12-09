package master

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type DBMgr struct {
	client *sqlx.DB
}

var (
	G_dbMgr *DBMgr
)

func InitDBMgr() (err error) {

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

	G_dbMgr = &DBMgr{
		client: db,
	}

	return

}


