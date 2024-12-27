package handler

import (
	"GOODS_SERVICE/biz/goods"
	"GOODS_SERVICE/proto"
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GoodsSrv struct {
	proto.UnimplementedGoodsServer
}

func (s *GoodsSrv) GetGoodsByRoom(ctx context.Context, req *proto.GetGoodsByRoomReq) (*proto.GoodsListResp, error) {
	// 参数处理
	fmt.Println(req.RoomId)
	if req.GetRoomId() <= 0 {
		// 无效的请求

		return nil, status.Error(codes.InvalidArgument, "方法1请求参数有误"+string(req.GetRoomId()))
	}
	// 去查询数据并封装返回的响应数据 --> 业务逻辑
	data, err := goods.GetGoodsByRoomId(ctx, req.GetRoomId())
	if err != nil {
		return nil, status.Error(codes.Internal, "内部错误")
	}
	return data, nil
}

func (s *GoodsSrv) GetGoodsDetails(ctx context.Context, req *proto.GetGoodsDetailReq) (*proto.GoodsDetailResp, error) {

	fmt.Println(req.GoodsId)
	if req.GetGoodsId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "方法2请求参数有误")
	}

	data, err := goods.GetGoodsDetails(ctx, req.GetGoodsId())
	if err != nil {
		return nil, status.Error(codes.Internal, "内部错误")
	}

	return data, nil
}
