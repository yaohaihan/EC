package handler

import (
	"STOCK_SERVICE/biz/stock"
	"STOCK_SERVICE/dao/mysql"
	"STOCK_SERVICE/model"
	"STOCK_SERVICE/proto"
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type StockSrv struct {
	proto.UnimplementedStockServer
}

func (s *StockSrv) GetStock(ctx context.Context, param *proto.GoodsStockInfo) (*proto.GoodsStockInfo, error) {
	data, err := stock.GetStockByGoodsId(ctx, param.GoodsId)

	if err != nil {
		zap.L().Error(
			"GetStock failed",
			zap.Int64("goods_id", param.GoodsId),
			zap.Error(err))
		return nil, status.Error(codes.Unimplemented, "something inside")
	}
	return data, nil
}

func (s *StockSrv) ReduceStock(ctx context.Context, param *proto.GoodsStockInfo) (*proto.GoodsStockInfo, error) {
	id := param.GoodsId
	num := param.Num
	orderId := param.OrderId
	if id <= 0 || num <= 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}
	data, err := stock.ReduceStockByGoodsId(ctx, id, num, orderId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to reduce stock")
	}

	return data, nil
}

func RollbackMsghandle(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		var data model.OrderGoodsStockInfo
		err := json.Unmarshal(msgs[i].Body, &data)
		if err != nil {
			zap.L().Error("json.Unmarshal RollbackMsg failed", zap.Error(err))
			continue
		}

		err = mysql.RollbackStockByMsg(ctx, data)

		if err != nil {
			return consumer.ConsumeRetryLater, nil
		}

		return consumer.ConsumeSuccess, nil
	}

	return consumer.ConsumeSuccess, nil
}
