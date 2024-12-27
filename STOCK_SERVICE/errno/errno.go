package errno

import "errors"

var (
	ErrQueryFailed = errors.New("query db failed") // 查询数据库失败

	ErrQueryEmpty = errors.New("query empty") // 查询结果为空

	ErrUnderstock = errors.New("understock") // 没有库存

	ErrReducestockFailed = errors.New("reduce stock failed") // 库存扣减失败

	ErrRollbackstockFailed = errors.New("rollback stock failed")
)
