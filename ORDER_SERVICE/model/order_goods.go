package model

type OrderDetail struct {
	BaseModel

	OrderId int64
	GoodsId int64
	UserId  int64
	Num     int64
}

func (OrderDetail) TableName() string {
	return "xx_order_detail"
}
