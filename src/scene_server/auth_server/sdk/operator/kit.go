/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package operator

import (
	"encoding/json"
	"fmt"
)

func isNumeric(val interface{}) bool {
	switch val.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		return true
	}
	return false
}

func toFloat64(val interface{}) float64 {
	switch vall := val.(type) {
	case int:
		return float64(vall)
	case int8:
		return float64(vall)
	case int16:
		return float64(vall)
	case int32:
		return float64(vall)
	case int64:
		return float64(vall)
	case uint:
		return float64(vall)
	case uint8:
		return float64(vall)
	case uint16:
		return float64(vall)
	case uint32:
		return float64(vall)
	case uint64:
		return float64(vall)
	case json.Number:
		val, _ := val.(json.Number).Float64()
		return val
	case float64:
		return vall
	case float32:
		return float64(vall)
	default:
		panic(fmt.Sprintf("unsupported type, value: %v", val))

	}
}
