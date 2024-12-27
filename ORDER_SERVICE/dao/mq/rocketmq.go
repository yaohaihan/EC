package mq

import (
	"ORDER_SERVICE/config"
	"ORDER_SERVICE/dao/mysql"
	"ORDER_SERVICE/model"
	"ORDER_SERVICE/proto"
	"ORDER_SERVICE/rpc"
	"context"
	"encoding/json"
	"fmt"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
)

var (
	Producer            rocketmq.Producer
	TransactionProducer rocketmq.TransactionProducer
)

func Init() (err error) {
	Producer, err = rocketmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{config.Conf.RocketMqConfig.Addr})),
		producer.WithRetry(2),
		producer.WithGroupName(config.Conf.RocketMqConfig.GroupId))

	if err != nil {
		fmt.Println(err)
		return err
	}
	err = Producer.Start()
	if err != nil {
		fmt.Println(err)
		return err
	}

	listener := &OrderEntity{}

	tp, err := rocketmq.NewTransactionProducer(listener,
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{config.Conf.RocketMqConfig.Addr})),
		producer.WithRetry(2),
		producer.WithGroupName("order_srv_1"))

	if err != nil {
		zap.L().Error("NewTransactionProducer failed", zap.Error(err))
		return status.Error(codes.Internal, "NewTransactionProducer failed")
	}

	if err := tp.Start(); err != nil {
		return fmt.Errorf("failed to start producer: %w", err)
	}

	TransactionProducer = tp

	return nil
}

func Exit() error {
	err := Producer.Shutdown()
	if err != nil {
		fmt.Printf("shutdown producer error: %s", err.Error())
	}
	return err
}

type OrderEntity struct { //implement two methods (ExecuteLocalTransaction and CheckLocalTransaction) so that orderEntity is a listener
	OrderId int64
	Param   *proto.OrderReq
	Err     error
}

// when prepare (half) message succeed this method will be ordered
func (o *OrderEntity) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	fmt.Println("in ExecuteLocalTransaction ... ")
	var param proto.OrderReq

	err := json.Unmarshal(msg.Body, &param)
	if err != nil {
		zap.L().Error("Failed to parse message body", zap.Error(err))
		return primitive.RollbackMessageState
	}
	//transactionID := msg.GetProperty(primitive.PropertyTransactionID)
	//
	//value, ok := TransactionParams.Load(transactionID)
	//if !ok {
	//	zap.L().Error("Transaction parameters not found")
	//	return primitive.RollbackMessageState
	//}

	//param, ok := value.(*proto.OrderReq)
	//
	//if !ok || param == nil {
	//	zap.L().Error("Invalid transaction parameters")
	//	return primitive.RollbackMessageState
	//}
	o.OrderId = param.OrderId
	o.Param = &param

	if o.Param == nil {
		zap.L().Error("ExecuteLocalTransaction param is nil")
		o.Err = status.Error(codes.Internal, "invalid OrderEntity")
		return primitive.CommitMessageState
	}

	ctx := context.Background()
	fmt.Println("before ")
	//1. rpc will connect to goods_service to get the price of goods
	goodsDetail, err := rpc.GoodsCli.GetGoodsDetails(ctx, &proto.GetGoodsDetailReq{
		GoodsId: param.GoodsId,
		UserId:  param.UserId,
	})
	fmt.Println("after")

	if err != nil {
		zap.L().Error("GoodsCli.GetGoodsDetail failed", zap.Error(err))
		// 库存未扣减
		o.Err = status.Error(codes.Internal, err.Error())
		return primitive.RollbackMessageState
	}

	payAmountStr := goodsDetail.Price
	payAmount, _ := strconv.ParseInt(payAmountStr, 10, 64)

	//stock verification and deduction  rpc connects to stock service
	_, err = rpc.StockCli.ReduceStock(ctx, &proto.GoodsStockInfo{
		OrderId: param.OrderId,
		GoodsId: param.GoodsId,
		Num:     param.Num,
	})

	if err != nil {
		// 库存扣减失败，丢弃half-message
		zap.L().Error("StockCli.ReduceStock failed", zap.Error(err))
		o.Err = status.Error(codes.Internal, "ReduceStock failed")
		return primitive.RollbackMessageState
	}

	// If the code execution reaches here, it means the inventory deduction was successful.
	// From here onward, if the local transaction fails, the inventory needs to be rolled back.
	// 3. Create Order
	// Generate order table
	orderData := model.Order{
		OrderId:        o.OrderId,
		UserId:         param.UserId,
		PayAmount:      payAmount,
		ReceiveAddress: param.Address,
		ReceiveName:    param.Name,
		ReceivePhone:   param.Phone,
		Status:         100, // 待支付
	}

	orderDetail := model.OrderDetail{
		OrderId: o.OrderId,
		UserId:  param.UserId,
		GoodsId: param.GoodsId,
		Num:     param.Num,
	}

	err = mysql.CreateOrderWithTransation(ctx, &orderData, &orderDetail)
	if err != nil {
		// 本地事务执行失败了，上一步已经库存扣减成功
		// 就需要将库存回滚的消息投递出去，下游根据消息进行库存回滚
		zap.L().Error("CreateOrderWithTransation failed", zap.Error(err))
		return primitive.CommitMessageState // 将之前发送的hal-message commit
	}

	// 发送延迟消息
	// 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h
	data := model.OrderGoodsStockInfo{
		OrderId: o.OrderId,
		GoodsId: param.GoodsId,
		Num:     param.Num,
	}
	b, _ := json.Marshal(data)
	//这个topic的目的是查看订单是否超时，查看其状态是否是已支付  因为这个demo项目并没有用到支付接口或者服务，所以这个topic只是展示一下逻辑
	msg = primitive.NewMessage(config.Conf.RocketMqConfig.Topic.PayTimeOut, b)
	msg.WithDelayTimeLevel(3)
	_, err = Producer.SendSync(context.Background(), msg)
	if err != nil {
		// 发送延时消息失败
		zap.L().Error("send delay msg failed", zap.Error(err))
		return primitive.CommitMessageState
	}

	// 走到这里说明 本地事务执行成功
	// 需要将之前的half-message rollback， 丢弃掉
	return primitive.RollbackMessageState
}

func (o *OrderEntity) CheckLocalTransaction(*primitive.MessageExt) primitive.LocalTransactionState {
	ctx := context.Background()
	_, err := mysql.QueryOrder(ctx, o.OrderId)
	if err != nil {
		return primitive.CommitMessageState
	}
	return primitive.RollbackMessageState

}
