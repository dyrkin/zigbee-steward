package coordinator

import (
	"github.com/dyrkin/znp-go"
	"reflect"
)

var SysResetIndType = reflect.TypeOf(&znp.SysResetInd{})
var ZdoActiveEpRspType = reflect.TypeOf(&znp.ZdoActiveEpRsp{})
var ZdoSimpleDescRspType = reflect.TypeOf(&znp.ZdoSimpleDescRsp{})
var ZdoNodeDescRspType = reflect.TypeOf(&znp.ZdoNodeDescRsp{})
var ZdoBindRspType = reflect.TypeOf(&znp.ZdoBindRsp{})
var ZdoUnbindRspType = reflect.TypeOf(&znp.ZdoUnbindRsp{})
