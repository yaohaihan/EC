package goods

import (
	"GOODS_SERVICE/dao/mysql"
	"GOODS_SERVICE/proto"
	"context"
	"encoding/json"
	"fmt"
)

func GetGoodsByRoomId(ctx context.Context, roomId int64) (*proto.GoodsListResp, error) {
	//先查es
	//再查mysql
	//更新redis

	//1. 先去 xx_room_goods 表 根据 room_id 查询出所有的 goods_id
	objList, err := mysql.GetGoodsByRoomId(ctx, roomId)
	if err != nil {
		return nil, err
	}

	//处理数据
	//1. 拿出所有的商品id
	//2. 记住当前正在讲解的商品id
	var (
		currGoodsId int64
		idList      = make([]int64, 0, len(objList))
	)

	for _, obj := range objList {
		idList = append(idList, obj.GoodsId)

		if obj.IsCurrent == 1 {
			currGoodsId = obj.GoodsId
		}
	}

	goodsList, err := mysql.GetGoodsById(ctx, idList)
	if err != nil {
		return nil, err
	}

	//得到商品列表之后还需要拼接proto的reponse，因为都是用grpc去沟通
	data := make([]*proto.GoodsInfo, 0, len(objList))
	for _, goods := range goodsList {
		var headImgs []string
		json.Unmarshal([]byte(goods.HeadImgs), &headImgs)
		data = append(data, &proto.GoodsInfo{
			GoodsId: goods.GoodsId,

			CategoryId:  goods.CategoryId,
			Status:      int32(goods.Status),
			Title:       goods.Title,
			MarketPrice: fmt.Sprintf("%.2f", float64(goods.MarketPrice/100)),
			Price:       fmt.Sprintf("%.2f", float64(goods.Price/100)),
			Brief:       goods.Brief,
			HeadImgs:    headImgs,
		})
	}

	resp := &proto.GoodsListResp{
		CurrentGoodsId: currGoodsId,
		Data:           data,
	}
	return resp, nil
}

func GetGoodsDetails(ctx context.Context, goodsId int64) (*proto.GoodsDetailResp, error) {
	goodsDetail, err := mysql.GetGoodsDetailById(ctx, goodsId)
	if err != nil {
		return nil, err
	}

	reponse := &proto.GoodsDetailResp{
		GoodsId:     goodsDetail.GoodsId,
		CategoryId:  goodsDetail.CategoryId,
		Status:      int32(goodsDetail.Status),
		Title:       goodsDetail.Title,
		MarketPrice: fmt.Sprintf("%.2f", float64(goodsDetail.MarketPrice/100)), //这一步是为了将int64转为string
		Price:       fmt.Sprintf("%.2f", float64(goodsDetail.Price)),
		Brief:       goodsDetail.Brief,
	}
	return reponse, nil
}
