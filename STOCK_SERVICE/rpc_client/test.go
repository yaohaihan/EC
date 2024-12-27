package main

import (
	"STOCK_SERVICE/proto"
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	conn   *grpc.ClientConn
	client proto.StockClient
)

// 创建RPC client端
func init() {
	var err error
	conn, err = grpc.Dial(
		"127.0.0.1:8382",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		panic(err)
	}
	client = proto.NewStockClient(conn)
}

// TestReduceStock 测试扣减库存服务
func TestReduceStock(wg *sync.WaitGroup) {
	defer wg.Done()
	param := &proto.GoodsStockInfo{
		GoodsId: 1,
		Num:     1,
	}
	resp, err := client.ReduceStock(context.Background(), param)
	fmt.Printf("resp:%v err:%v\n", resp, err)
}

func main() {
	defer conn.Close()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go TestReduceStock(&wg)
		//go f1(&wg)
	}
	wg.Wait()
	param := &proto.GoodsStockInfo{
		GoodsId: 1,
		Num:     1,
	}
	fmt.Println(client.GetStock(context.Background(), param))
}

var num int = 100

func f1(wg *sync.WaitGroup) {
	defer wg.Done()
	num = num - 1
}
