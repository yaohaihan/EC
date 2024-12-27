package mysql

import (
	"GOODS_SERVICE/errno"
	"GOODS_SERVICE/model"
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// dao 层用来执行数据库相关的操作

func GetGoodsByRoomId(ctx context.Context, roomId int64) ([]*model.RoomGoods, error) {
	var data []*model.RoomGoods

	err := db.WithContext(ctx).
		Model(&model.RoomGoods{}).
		Where("room_id = ?", roomId).
		Order("weight").
		Find(&data).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errno.ErrQueryFailed
	}
	return data, nil
}

func GetGoodsById(ctx context.Context, idList []int64) ([]*model.Goods, error) {
	var data []*model.Goods
	err := db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("goods_id in ?", idList).
		Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: "FIELD(goods_id,?)", Vars: []interface{}{idList}, WithoutParentheses: true},
		}).
		Find(&data).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errno.ErrQueryFailed
	}
	return data, nil
}

func GetGoodsDetailById(ctx context.Context, id int64) (*model.Goods, error) {
	var data model.Goods
	err := db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("goods_id = ?", id).
		First(&data).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errno.ErrQueryFailed
	}

	return &data, nil
}
