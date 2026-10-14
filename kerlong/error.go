package kerlong

import (
	"github.com/KouKouChan/CSO2-Server/logging"
)

// CheckErr 有错误就显示
func CheckErr(err error) {
	if err != nil {
		logging.Error(err)
	}
}
