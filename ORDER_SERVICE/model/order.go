package model

type Order struct {
	BaseModel

	OrderId   int64
	UserId    int64
	PayAmount int64
	Status    int32

	ReceiveAddress string
	ReceiveName    string
	ReceivePhone   string
}

func (Order) TableName() string {
	return "xx_order"
}
