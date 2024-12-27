package mysql

import (
	"STOCK_SERVICE/dao/redis"
	"STOCK_SERVICE/errno"
	"STOCK_SERVICE/model"
	"context"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// dao 层用来执行数据库相关的操作

func GetStockByGoodsId(ctx context.Context, goodsId int64) (*model.Stock, error) {
	var data model.Stock

	err := db.WithContext(ctx).
		Model(&model.Stock{}).          //利用stock里面的TableName去指定表名
		Where("goods_id = ?", goodsId). //查询条件
		First(&data).                   //这里一定要用data的指针
		Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errno.ErrQueryFailed
	}
	return &data, nil
}

//func ReduceStock(ctx context.Context, goodsId int64, num int64) (*model.Stock, error) {
//	var data model.Stock
//	err := db.WithContext(ctx).
//		Model(&model.Stock{}).
//		Where("goods_id = ?", goodsId).
//		First(&data).Error
//
//	if err != nil {
//		return nil, err
//	}
//
//	if data.Num-num < 0 {
//		return nil, errno.ErrUnderstock
//	}
//
//	data.Num = data.Num - num
//	//err = db.WithContext(ctx).Save(&data).Error
//	//err = db.WithContext(ctx).Model(&data).UpdateColumn("num", data.Num).Error
//	err = db.WithContext(ctx).Model(&model.Stock{}).Where("goods_id = ?", goodsId).UpdateColumn("num", data.Num).Error
//	if err != nil {
//		zap.L().Error(
//			"reduceStock save failed",
//			zap.Int64("goods_id", goodsId))
//		return nil, err
//	}
//	return &data, nil
//}

// 悲观锁,必须搭配事务使用,clause会锁定选取的行,不允许其他操作来修改它,从而实现悲观锁
//func ReduceStock(ctx context.Context, goodsId int64, num int64) (*model.Stock, error) {
//	var data model.Stock
//
//	db.Transaction(func(tx *gorm.DB) error {
//		err := tx.WithContext(ctx).
//			Clauses(clause.Locking{Strength: "UPDATE"}). // 添加 FOR UPDATE 子句
//			Where("goods_id = ?", goodsId).
//			First(&data).Error
//
//		if err != nil {
//			return err
//		}
//
//		if data.Num-num < 0 {
//			return errno.ErrUnderstock
//		}
//
//		data.Num = data.Num - num
//
//		err = tx.WithContext(ctx).Model(&model.Stock{}).Where("goods_id = ?", goodsId).UpdateColumn("num", data.Num).Error
//
//		if err != nil {
//			zap.L().Error(
//				"reduceStock save failed",
//				zap.Int64("goods_id", goodsId),
//			)
//			tx.Rollback()
//			return err
//		}
//		return nil
//	})
//	return &data, nil
//}

// 乐观锁版本
//func ReduceStock(ctx context.Context, goodsId int64, num int64) (*model.Stock, error) {
//	var (
//		data      model.Stock
//		retry     = 0
//		isSuccess = false
//	)
//
//	for !isSuccess && retry < 20 {
//		err := db.WithContext(ctx).
//			Where("goods_id = ?", goodsId).
//			First(&data).Error
//
//		if err != nil {
//			return nil, err
//		}
//
//		if data.Num-num < 0 {
//			return nil, errno.ErrUnderstock
//		}
//
//		//3.扣减
//		data.Num = data.Num - num
//
//		//这里用行数好一点,去判断是否得到返回值,n等于0说明version有变化所以需要继续循环
//		//当然也可以用err,去判断err是否为nil
//		n := db.WithContext(ctx).
//			Model(&model.Stock{}).
//			Where("goods_id = ? and version = ?", data.GoodsId, data.Version).
//			Updates(map[string]interface{}{
//				"goods_id": data.GoodsId,
//				"num":      data.Num,
//				"version":  data.Version + 1,
//			}).RowsAffected
//
//		if n < 1 {
//			fmt.Printf("update err:%v\n", err)
//			retry++ // 更新失败就重试
//			continue
//		}
//
//		isSuccess = true
//		break
//	}
//
//	if isSuccess {
//		return &data, nil
//	}
//
//	return nil, errno.ErrReducestockFailed
//}

// 分布式版本
func ReduceStock(ctx context.Context, goodsId, num, orderId int64) (*model.Stock, error) {
	var data model.Stock

	//先上锁然后再开启事务，如果反着来，先开始事务然后上锁会导致开启多个事务，造成资源浪费甚至事务冲突
	mutexname := fmt.Sprintf("xx-stock-%d", goodsId)
	//获取锁
	mutex := redis.Rs.NewMutex(mutexname)

	if err := mutex.Lock(); err != nil {
		return nil, errno.ErrReducestockFailed
	}
	// 此时data可能都是旧数据 data.num = 99  实际上数据库中num=97
	defer mutex.Unlock() // 释放锁
	// 获取锁成功
	// 开启事务

	db.Transaction(func(tx *gorm.DB) error {
		//获取锁id
		err := tx.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}). // 添加 FOR UPDATE 子句
			Where("goods_id = ?", goodsId).
			First(&data).Error

		if err != nil {
			return err
		}

		if data.Num-num < 0 {
			return errno.ErrUnderstock
		}

		data.Num = data.Num - num
		data.Lock = data.Lock + num

		err = tx.WithContext(ctx).Model(&model.Stock{}).Where("goods_id = ?", goodsId).UpdateColumn("num", data.Num).UpdateColumn("lock", data.Lock).Error

		if err != nil {
			zap.L().Error(
				"reduceStock save failed",
				zap.Int64("goods_id", goodsId),
			)
			tx.Rollback()
			return err
		}

		stockRecord := model.StockRecord{
			OrderId: orderId,
			GoodsId: goodsId,
			Num:     num,
			Status:  1, // 预扣减
		}

		err = tx.WithContext(ctx).
			Model(&model.StockRecord{}).
			Create(&stockRecord).Error
		if err != nil {
			zap.L().Error("create StockRecord failed", zap.Error(err))
			return err
		}

		return nil
	})
	return &data, nil
}

func RollbackStockByMsg(ctx context.Context, data model.OrderGoodsStockInfo) error {
	//firstly query stock data, this needs to be included in the transaction operation
	db.Transaction(func(tx *gorm.DB) error {
		var sr model.StockRecord
		err := tx.WithContext(ctx).
			Model(&model.StockRecord{}).
			Where("order_id = ? and goods_id = ? and status = 1", data.OrderId, data.GoodsId).
			First(&sr).Error

		// Record not found
		// Either the record doesn't exist or it has already been rolled back; no further action is needed.
		if err == gorm.ErrRecordNotFound {
			zap.L().Error("query stock_record by order_id failed", zap.Error(err), zap.Int64("order_id", data.OrderId), zap.Int64("goods_id", data.GoodsId))
			return err
		}

		//execution reaches here, meaning stock does need rollback
		var s model.Stock
		err = tx.WithContext(ctx).Model(&model.Stock{}).Where("goods_id = ?", data.GoodsId).First(&s).Error
		if err == gorm.ErrRecordNotFound {
			zap.L().Error("query stock by goods_id failed", zap.Error(err), zap.Int64("goods_id", data.GoodsId))
			return err
		}

		s.Num = s.Num + data.Num
		s.Lock -= data.Num // 锁定的库存减掉

		if s.Lock < 0 {
			return errno.ErrRollbackstockFailed
		}

		err = tx.WithContext(ctx).Save(&s).Error
		if err != nil {
			zap.L().Warn("RollbackStock stock save failed", zap.Int64("goods_id", s.GoodsId), zap.Error(err))
			return err
		}

		sr.Status = 3
		err = tx.WithContext(ctx).Save(&sr).Error
		if err != nil {
			zap.L().Warn("RollbackStock stock_record save failed", zap.Int64("goods_id", s.GoodsId), zap.Error(err))
			return err
		}

		return nil
	})
	return nil
}
