package mysql

import (
	"GOODS_SERVICE/config"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"time"
)

var db *gorm.DB

func Init(cfg *config.MySQLConfig) (err error) {
	//dsn 是 Data Source Name，表示数据库连接的配置信息。
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DB)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	//gorm.Open 用来初始化 Gorm 数据库连接。
	//mysql.Open(dsn) 表示使用 MySQL 驱动连接数据库。
	//&gorm.Config{} 是 Gorm 的全局配置，通常可以用于自定义 Gorm 的行为。

	if err != nil {
		return err
	}

	// 额外的连接配置
	sqlDB, err := db.DB()
	if err != nil {
		return
	}

	// 以下配置要配合 my.conf 进行配置
	// SetMaxIdleConns 设置空闲连接池中连接的最大数量
	sqlDB.SetMaxOpenConns(cfg.MaxIdleConns)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxIdleConns(cfg.MaxOpenConns)

	// SetConnMaxLifetime 设置了连接可复用的最大时间。
	sqlDB.SetConnMaxLifetime(time.Hour)
	return
}
