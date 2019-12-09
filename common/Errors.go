package common

import "errors"

var(
	ERR_LOCK_ALREADY_REQUIRED = errors.New("锁已被占用")
	ERR_NO_LOCAL_IP_FOUND = errors.New("没有找到本地IP")
	ERR_ACCOUNT_NOT_FOUND = errors.New("账户不存在")
)

