package metadata

import (
	"configcenter/src/common/util"
	"encoding/json"
	"fmt"
	"math"

	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/bson"
)

type ArrayOption[T any] struct {
	Len    int `bson:"len" json:"len" `
	Cap    int `bson:"cap" json:"cap" `
	Option T   `bson:"option" json:"option" `
}

func (a *ArrayOption[T]) Valid() error {
	if a.Len < 0 || a.Len > a.Cap {
		return fmt.Errorf("invalid array option,len:%d cap:%d", a.Len, a.Cap)
	}
	return nil
}

func ParseArrayOption[T any](option any) (ArrayOption[T], error) {
	if option == nil || option == "" {
		return ArrayOption[T]{Len: math.MaxInt, Cap: math.MaxInt}, nil
	}

	var result ArrayOption[T]

	var optMap map[string]interface{}
	switch value := option.(type) {
	case ArrayOption[T]:
		return value, nil
	case bson.M:
		optMap = value
	case map[string]interface{}:
		optMap = value
	default:
		marshal, err := json.Marshal(option)
		if err != nil {
			return result, fmt.Errorf("invalid array option,type:%v,value:%v,err:%w", option, option, err)
		}
		lenItem := gjson.GetBytes(marshal, "len")
		capItem := gjson.GetBytes(marshal, "cap")
		if !lenItem.Exists() || !lenItem.Exists() {
			return result, fmt.Errorf("invalid array option,type:%v,value:%v,err: not exist len or cap", option, option)
		}
		result.Len = int(capItem.Int())
		result.Cap = int(capItem.Int())
		return result, result.Valid()
	}
	lenn, lenOk := optMap["len"]
	capp, capOk := optMap["cap"]
	if !lenOk || !capOk {
		return result, fmt.Errorf("invalid array option,type:%v,value:%v,err: not exist len or cap", option, option)
	}
	capOpt, err := util.GetIntByInterface(capp)
	if err != nil {
		return result, err
	}
	result.Cap = capOpt
	lenOpt, err := util.GetIntByInterface(lenn)
	if err != nil {
		return result, err
	}
	result.Len = lenOpt
	return result, result.Valid()
}
