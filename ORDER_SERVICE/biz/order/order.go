package order

import (
	"ORDER_SERVICE/config"
	"ORDER_SERVICE/dao/mq"
	"ORDER_SERVICE/dao/mysql"
	"ORDER_SERVICE/model"
	"ORDER_SERVICE/proto"
	"ORDER_SERVICE/rpc"
	"ORDER_SERVICE/third_pkg/snowflake"
	"context"
	"encoding/json"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
)

func CreateOrder(ctx context.Context, param *proto.OrderReq) error {
	//0. generate orderId
	orderId := snowflake.GenID()

	//1. query product price   --> rpc call
	//for each request, fetch a avaliable product service address from Consul
	//get the address to create rpc client and execute the rpc call
	goodsDetail, err := rpc.GoodsCli.GetGoodsDetails(ctx, &proto.GetGoodsDetailReq{
		GoodsId: param.GoodsId,
		UserId:  param.UserId,
	})

	if err != nil {
		zap.L().Error("GoodsCli.GetGoodsDetail failed", zap.Error(err))
		return err
	}

	payAmountStr := goodsDetail.Price
	payAmount, _ := strconv.ParseInt(payAmountStr, 10, 64)

	//3. inventory verification and deduction
	// For each request, fetch an available inventory service address from Consul
	// Use the address to create an RPC client and perform the RPC call
	_, err = rpc.StockCli.ReduceStock(ctx, &proto.GoodsStockInfo{
		GoodsId: param.GoodsId,
		Num:     param.Num,
	})

	if err != nil {
		zap.L().Error("StockCli.ReduceStock failed", zap.Error(err))
		return err
	}

	//4. create order wrap both both create order and creeate order info using transaction
	orderData := model.Order{
		OrderId:        orderId,
		UserId:         param.UserId,
		PayAmount:      payAmount,
		ReceiveAddress: param.Address,
		ReceiveName:    param.Name,
		ReceivePhone:   param.Phone,
	}

	//4.2 创建订单详情表
	orderDetail := model.OrderDetail{
		OrderId: orderId,
		UserId:  param.UserId,
		GoodsId: param.GoodsId,
		Num:     param.Num,
	}

	return mysql.CreateOrderWithTransation(ctx, &orderData, &orderDetail)

}

func Create(ctx context.Context, param *proto.OrderReq) error {
	orderId := snowflake.GenID()

	orderEntity := &mq.OrderEntity{
		OrderId: orderId,
		Param:   param,
		Err:     nil,
	}

	//encapsulate message: orderId goodsId num
	data := model.OrderGoodsStockInfo{ //发送消息时，需要告诉买了订单id 哪个商品 买了多少件
		OrderId: orderId,
		GoodsId: param.GoodsId,
		Num:     param.Num,
	}

	b, _ := json.Marshal(data)

	msg := &primitive.Message{
		Topic: config.Conf.RocketMqConfig.Topic.StockRollback,
		Body:  b,
	}

	// send transaction message
	res, err := mq.TransactionProducer.SendMessageInTransaction(ctx, msg) //
	if err != nil {
		zap.L().Error("SendMessageInTransaction failed", zap.Error(err))
		return status.Error(codes.Internal, "create order failed")
	}
	zap.L().Info("p.SendMessageInTransaction success", zap.Any("res", res))

	//CommitMessageState means that localtransaction fails
	if res.State == primitive.CommitMessageState {
		return status.Error(codes.Aborted, "create order failed")
	}

	// 其他内部错误
	if orderEntity.Err != nil {
		return orderEntity.Err
	}
	return nil
}
