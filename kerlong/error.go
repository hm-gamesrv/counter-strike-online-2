package kerlong

import (
	"cso2-server/logging"
)

// CheckErr 有错误就显示
func CheckErr(err error) {
	if err != nil {
		logging.Error(err)
	}
}
