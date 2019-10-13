package core

import (
	"context"
	"database/sql"
	"errors"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"os"
	"strconv"
	"time"
)

type Mysql struct {
	db *sql.DB
}

var GMysql *Mysql

func InitMysql() error {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_DSN"))
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)
	GMysql = &Mysql{db: db}
	return nil
}

// id 为64位   step 最大不超过1000000
func (mysql *Mysql) NextId(bizTag string, step int64) (maxId int64, datastep int64, err error) {
	var (
		tx           *sql.Tx
		query        string
		stmt         *sql.Stmt
		result       sql.Result
		rowsAffected int64
	)

	// 总耗时小于2秒
	ctx, cancelFunc := context.WithTimeout(context.Background(), time.Duration(2000)*time.Millisecond)
	defer cancelFunc()

	// 开启事务
	if tx, err = mysql.db.BeginTx(ctx, nil); err != nil {
		return
	}

	// 1. 前进一个步长, 即占用一个号段(更新操作是悲观行锁)
	// 2. MySQL的update语句，set列的顺序是有关系的，后面列的计算是以前面列的结果为基础的
	if step == 0 {
		query = "UPDATE segments SET max_id=max_id+step WHERE biz_tag=?"
	} else {
		query = "UPDATE segments SET step=" + strconv.FormatInt(step, 10) + ",max_id=max_id+step WHERE biz_tag=?"
	}

	if stmt, err = tx.PrepareContext(ctx, query); err != nil {
		goto ROLLBACK
	}

	if result, err = stmt.ExecContext(ctx, bizTag); err != nil {
		goto ROLLBACK
	}

	if rowsAffected, err = result.RowsAffected(); err != nil { // 失败
		goto ROLLBACK
	} else if rowsAffected == 0 { // 记录不存在
		err = errors.New("biz_tag not found")
		goto ROLLBACK
	}

	// 2, 查询更新后的最新max_id, 此时仍在事务内, 行锁保护下
	query = "SELECT max_id, step FROM segments WHERE biz_tag=?"
	if stmt, err = tx.PrepareContext(ctx, query); err != nil {
		goto ROLLBACK
	}
	if err = stmt.QueryRowContext(ctx, bizTag).Scan(&maxId, &datastep); err != nil {
		goto ROLLBACK
	}

	// 3, 提交事务
	err = tx.Commit()
	log.Printf("数据库step：%v", datastep)
	return

ROLLBACK:
	tx.Rollback()
	return
}
