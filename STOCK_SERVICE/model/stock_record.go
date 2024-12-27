package model

type StockRecord struct {
	BaseModel

	OrderId int64
	GoodsId int64
	Num     int64
	Status  int32
}

func (StockRecord) TableName() string {
	return "xx_stock_record"
}
