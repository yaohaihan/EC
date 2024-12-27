package main

import (
	"GOODS_SERVICE/config"
	"GOODS_SERVICE/dao/mysql"
	"GOODS_SERVICE/handler"
	"GOODS_SERVICE/logger"
	"GOODS_SERVICE/proto"
	"GOODS_SERVICE/registry"
	"context"
	"flag"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	//1.加载配置  cfn means configuration filename
	var cfn string
	flag.StringVar(&cfn, "conf", "./conf/config.yaml", "指定配置文件路径")
	flag.Args()
	flag.Parse()

	err := config.Init(cfn)
	if err != nil {
		panic(err) //程序启动时加载配置文件失败
	}

	//2.加载日志
	err = logger.Init(config.Conf.LogConfig, config.Conf.Mode)
	if err != nil {
		panic(err)
	}

	//3.加载mysql
	err = mysql.Init(config.Conf.MySQLConfig)
	if err != nil {
		panic(err)
	}

	//4.加载Consul

	err = registry.Init(config.Conf.ConsulConfig.Addr)
	if err != nil {
		panic(err)
	}

	//监听端口  lis是一个网络端口监听器，监听的是来自客户端的链接
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.RpcPort)) //这里只绑定了端口，没指定ip就会绑定所有网卡的对应端口，内网的外网的都会绑定，然后因为consul是容器去运行的和windows是隔离开的，所以

	if err != nil {
		panic(err)
	}

	//创建grpc服务
	s := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(s, health.NewServer())

	//注册商品的rpc服务
	proto.RegisterGoodsServer(s, &handler.GoodsSrv{})

	//启动gRPC服务
	go func() {
		err = s.Serve(lis) //让启动的这个grpc服务去阶段lis这个，这个lis监听器就相当于门，通过这哥们可以监听到客户端的请求
		if err != nil {
			panic(err)
		}
	}()

	//。它将启动的 gRPC 服务信息（例如服务名称、IP、端口等）注册到一个consul中
	registry.Reg.RegisterService(config.Conf.Name, config.Conf.IP, config.Conf.RpcPort, nil)

	zap.L().Info(
		"rpc server start",
		zap.String("ip", config.Conf.IP),
		zap.Int("port", config.Conf.RpcPort),
	)

	//然后是启动grpc gateway
	//1. 先搞出来个连接
	// Create a client connection to the gRPC server we just started
	// This is where the gRPC-Gateway proxies the requests
	conn, err := grpc.DialContext(
		context.Background(),
		fmt.Sprintf("%s:%d", config.Conf.IP, config.Conf.RpcPort), //指定了对应ip和端口上的grpc服务，和他创建连接
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("Failed to dial server", err)
	}

	//2. 初始化 gRPC-Gateway 的 HTTP 路由器，用来将 HTTP 请求转发到 gRPC 服务。
	//runtime.NewServeMux() 创建了一个支持 JSON 编码和解码的多路复用器。
	gwmux := runtime.NewServeMux()

	//3. 用前两步骤搞出来的东西 将 Goods 服务注册到 gRPC-Gateway。
	err = proto.RegisterGoodsHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatalln("Failed to register gateway", err)
	}

	//4.创建并启动 HTTP 服务，用第三步搞出来的去
	gwServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Conf.HttpPort),
		Handler: gwmux,
	}

	zap.S().Infof("Serving gRPC-Gateway on http://0.0.0.0:%d", config.Conf.HttpPort)

	go func() {
		err = gwServer.ListenAndServe()
		if err != nil {
			log.Printf("Failed to listen and serve port 8091: %v", err)
			return
		}
	}()

	//服务退出时要注销服务
	//创建一个专门用来接收系统信号的通道
	quit := make(chan os.Signal)

	//就是让出现SIGTERM  SIGQUIT这两种信号时，放进 quit通道
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	//退出时注销服务
	serviceId := fmt.Sprintf("%s-%s-%s", config.Conf.IP, config.Conf.RpcPort)
	registry.Reg.Deregister(serviceId)
}
