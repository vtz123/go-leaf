package core

import (
	"errors"
	"github.com/joho/godotenv"
	"log"
)

func Init() error {
	//配置初始化
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	//数据库初始化
	if err = InitMysql(); err != nil {
		log.Print(err)
	}
	//模式初始化
	if err = InitAllocSegment(); err != nil {
		err = errors.New("Segment模式 初始化错误")
		log.Print(err)
	}
	if err = InitAllocSnowflake(); err != nil {
		err = errors.New("Segment模式 初始化错误")
		log.Print(err)
	}
	return err
}
