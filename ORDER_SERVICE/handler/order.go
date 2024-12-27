package handler

import (
	"ORDER_SERVICE/biz/order"
	"ORDER_SERVICE/config"
	"ORDER_SERVICE/dao/mq"
	"ORDER_SERVICE/dao/mysql"
	"ORDER_SERVICE/model"
	"ORDER_SERVICE/proto"
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type OrderSrv struct {
	proto.UnimplementedOrderServer
}

func (s *OrderSrv) CreateOrder(ctx context.Context, req *proto.OrderReq) (*emptypb.Empty, error) {
	fmt.Println("in CreateOrder ... ")

	if req.GetUserId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "请求参数有误")
	}

	err := order.Create(ctx, req)

	if err != nil {
		zap.L().Error("order.Create failed", zap.Error(err))
		return nil, status.Error(codes.Internal, "内部错误")
	}

	return &emptypb.Empty{}, nil
}

func OrderTimeoutHandle(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		var data model.OrderGoodsStockInfo
		err := json.Unmarshal(msgs[i].Body, &data)
		if err != nil {
			zap.L().Error("json.Unmarshal RollbackMsg failed", zap.Error(err))
			continue
		}

		//query order
		// 1. 如果订单为已支付状态则不处理
		// 2. 如果订单为未支付状态则发送一条回滚库存的消息
		o, err := mysql.QueryOrder(ctx, data.OrderId)
		if err != nil {
			zap.L().Error("mysql.QueryOrder failed", zap.Error(err))
			return consumer.ConsumeRetryLater, nil // 稍后再试
		}

		if o.OrderId == data.OrderId && o.Status == 100 {
			msg := &primitive.Message{
				Topic: config.Conf.RocketMqConfig.Topic.StockRollback,
				Body:  msgs[i].Body,
			}

			_, err = mq.Producer.SendSync(context.Background(), msg)
			if err != nil {
				zap.L().Error("send rollback msg failed", zap.Error(err))
				return consumer.ConsumeRetryLater, nil // 稍后再试
			}

			o.Status = 300
			mysql.UpdateOrder(ctx, o)
		}
	}
	return consumer.ConsumeSuccess, nil
}

//func OrderTimeoutHandleBack(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
//	for i := range msgs {
//		var data model.OrderGoodsStockInfo
//
//		err := json.Unmarshal(msgs[i].Body, &data)
//		if err != nil {
//			zap.L().Error("mysql.QueryOrder failed", zap.Error(err))
//			continue
//		}
//
//		o, err := mysql.QueryOrder(ctx, data.OrderId)
//		if err != nil {
//			zap.L().Error("mysql.QueryOrder failed", zap.Error(err))
//			return consumer.ConsumeRetryLater, nil // 稍后再试
//		}
//
//		if o.Status ==  {
//
//		}
//
//	}
//}
