package server

import "fmt"

// nolint: golint
var (
	OK                    = &Errno{Code: 0, Message: "OK"}
	InternalServerError   = &Errno{Code: 10001, Message: "Internal server error"}
	ErrToken              = &Errno{Code: 10002, Message: "token错误"}
	ErrParam              = &Errno{Code: 10003, Message: "参数有误"}
	ErrNotData            = &Errno{Code: 10004, Message: "没有数据"}
	ErrNotChangeData      = &Errno{Code: 10005, Message: "数据没有更改"}
	ErrNotRepeatData      = &Errno{Code: 10006, Message: "数据已存在"}
	ErrEngine             = &Errno{Code: 10007, Message: "Engine Not"}
	ErrCreateWallet       = &Errno{Code: 10008, Message: "创建钱包失败"}
	ErrNoAddress          = &Errno{Code: 10009, Message: "未传入地址"}
	ErrNoSuccess          = &Errno{Code: 10010, Message: "交易未成功"}
	ErrSignGroupLengthErr = &Errno{Code: 10011, Message: "多签授权数组地址错误"}
	ErrWalletNotInDB      = &Errno{Code: 10012, Message: "地址不在数据库中"}
	ErrPasswdErr          = &Errno{Code: 10013, Message: "密码错误"}
	ErrAccountErr         = &Errno{Code: 10014, Message: "账户错误"}
	ErrLoginExpire        = &Errno{Code: 10015, Message: "登录超时"}
	ErrNoPremission       = &Errno{Code: 10016, Message: "权限不允许"}
	ErrSame20Token        = &Errno{Code: 10017, Message: "存在相同代币"}
)

// Errno ...
type Errno struct {
	Code    int
	Message string
}

func (err Errno) Error() string {
	return err.Message
}

// Err represents an error
type Err struct {
	Code    int
	Message string
	Err     error
}

func (err *Err) Error() string {
	return fmt.Sprintf("Err - code: %d, message: %s, error: %s", err.Code, err.Message, err.Err)
}

// DecodeErr ...
func DecodeErr(err error) (int, string) {
	if err == nil {
		return OK.Code, OK.Message
	}

	switch typed := err.(type) {
	case *Err:
		return typed.Code, typed.Message
	case *Errno:
		return typed.Code, typed.Message
	default:
	}

	return InternalServerError.Code, err.Error()
}
