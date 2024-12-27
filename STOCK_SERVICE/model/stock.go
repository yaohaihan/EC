package model

type Stock struct {
	BaseModel
	GoodsId int64
	Num     int64
	Lock    int64
}

func (Stock) TableName() string {
	return "xx_stock"
}
