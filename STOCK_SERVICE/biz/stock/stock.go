package stock

import (
	"STOCK_SERVICE/dao/mysql"
	"STOCK_SERVICE/errno"
	"STOCK_SERVICE/proto"
	"context"
)

func GetStockByGoodsId(ctx context.Context, goodsId int64) (*proto.GoodsStockInfo, error) {

	data, err := mysql.GetStockByGoodsId(ctx, goodsId)
	if err != nil {
		return nil, err
	}

	resp := &proto.GoodsStockInfo{
		GoodsId: data.GoodsId,
		Num:     data.Num,
	}

	return resp, nil
}

func ReduceStockByGoodsId(ctx context.Context, goodsId int64, num int64, orderId int64) (*proto.GoodsStockInfo, error) {
	data, err := mysql.ReduceStock(ctx, goodsId, num, orderId)
	if err != nil {
		return nil, errno.ErrUnderstock
	}

	return &proto.GoodsStockInfo{
		GoodsId: data.GoodsId,
		Num:     data.Num,
	}, nil
}
