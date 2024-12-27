package rpc

import (
	"ORDER_SERVICE/config"
	"ORDER_SERVICE/proto"
	"errors"
	"fmt"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

const (
	MsgType = iota
	MsgType2
	MsgType3
	MsgType4
)

var (
	GoodsCli proto.GoodsClient
	StockCli proto.StockClient
)

func InitSrvClientBack() error {
	if len(config.Conf.GoodsService.Name) == 0 {
		return errors.New("invalid GoodsService.Name")
	}
	if len(config.Conf.StockService.Name) == 0 {
		return errors.New("invalid StockService.Name")
	}

	// consul实现服务发现
	// 程序启动的时候请求consul获取一个可以用的商品服务地址
	zap.L().Info("Connecting to Consul",
		zap.String("Consul Address", config.Conf.ConsulConfig.Addr),
		zap.String("Service Name", config.Conf.GoodsService.Name),
	)
	goodsConn, err := grpc.Dial(
		fmt.Sprintf("consul://%s/%s?wait=14s", config.Conf.ConsulConfig.Addr, config.Conf.GoodsService.Name),
		// 指定round_robin策略
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("dial goods_srv failed, err:%v\n", err)
		return err
	}
	GoodsCli = proto.NewGoodsClient(goodsConn)

	stockConn, err := grpc.Dial(
		// consul服务
		fmt.Sprintf("consul://%s/%s?wait=14s", config.Conf.ConsulConfig.Addr, config.Conf.StockService.Name),
		// 指定round_robin策略
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("dial stock_srv failed, err:%v\n", err)
		return err
	}
	StockCli = proto.NewStockClient(stockConn)
	return nil
}

func InitSrvClient() error {
	if len(config.Conf.GoodsService.Name) == 0 {
		return errors.New("invalid GoodsService.Name")
	}
	if len(config.Conf.StockService.Name) == 0 {
		return errors.New("invalid StockService.Name")
	}

	consulClient, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}

	goodsAddress, err := GetServiceAddress(consulClient, config.Conf.GoodsService.Name)
	if err != nil {
		log.Fatalf("Failed to get service address for %s: %v", config.Conf.GoodsService.Name, err)
	}

	zap.L().Info("Connecting to GoodsService", zap.String("Consul Address", goodsAddress))

	goodsConn, err := grpc.Dial(goodsAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", config.Conf.GoodsService.Name, err)
	}

	GoodsCli = proto.NewGoodsClient(goodsConn)

	stockAddress, err := GetServiceAddress(consulClient, config.Conf.StockService.Name)

	if err != nil {
		log.Fatalf("Failed to get service address for %s: %v", config.Conf.StockService.Name, err)
	}
	zap.L().Info("Connecting to StockService", zap.String("Consul Address", stockAddress))

	stockConn, err := grpc.Dial(
		// consul服务
		stockAddress,
		// 指定round_robin策略
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy": "round_robin"}`),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("dial stock_srv failed, err:%v\n", err)
		return err
	}
	StockCli = proto.NewStockClient(stockConn)
	return nil
}

func GetServiceAddress(client *api.Client, serviceName string) (string, error) {
	// 使用健康检查接口获取服务实例
	services, _, err := client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return "", err
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no healthy instance found for service: %s", serviceName)
	}

	// 获取第一个健康实例的地址和端口
	service := services[0]
	address := fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
	return address, nil
}
