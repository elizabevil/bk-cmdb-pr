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

package middleware

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	httpheader "configcenter/src/common/http/header"
	"configcenter/src/common/metadata"
	"github.com/tidwall/gjson"
)

const (
	cacheTime = time.Minute * 5
)

const (
	defaultRelateAttr = "host"
)

func (d *Discover) parseData(msg *string) (data map[string]interface{}, err error) {
	dataStr := gjson.Get(*msg, "data.data").String()
	if err = json.Unmarshal([]byte(dataStr), &data); err != nil {
		blog.Errorf("parse data error: %s", err)
		return
	}
	return
}

func (d *Discover) parseObjID(msg *string) string {
	return gjson.Get(*msg, "data.meta.model.bk_obj_id").String()
}

func (d *Discover) parseOwnerId(msg *string) string {
	ownerId := gjson.Get(*msg, "data.meta.model.bk_supplier_account").String()

	if ownerId == "" {
		ownerId = common.BKDefaultOwnerID
	}
	return ownerId
}

// CreateInstKey TODO
func (d *Discover) CreateInstKey(objID string, ownerID string, val []string) string {
	return fmt.Sprintf("cc:v3:inst[%s:%s:%s:%s]",
		common.CCSystemCollectorUserName,
		ownerID,
		objID,
		strings.Join(val, ":"),
	)
}

// GetInstFromRedis TODO
func (d *Discover) GetInstFromRedis(instKey string) (map[string]interface{}, error) {

	val, err := d.redisCli.Get(d.ctx, instKey).Result()
	if err != nil {
		return nil, fmt.Errorf("%s: get inst cache error: %s", instKey, err)
	}

	var cacheData = make(map[string]interface{})
	err = json.Unmarshal([]byte(val), &cacheData)
	if err != nil {
		return nil, fmt.Errorf("marshal condition error: %s", err)
	}

	return cacheData, nil

}

// TrySetRedis TODO
func (d *Discover) TrySetRedis(key string, value []byte, duration time.Duration) {
	_, err := d.redisCli.Set(d.ctx, key, value, duration).Result()
	if err != nil {
		blog.Warnf("%s: flush to redis failed: %s", key, err)
	} else {

		blog.Infof("%s: flush to redis success", key)
	}
}

// TryUnsetRedis TODO
func (d *Discover) TryUnsetRedis(key string) {
	_, err := d.redisCli.Del(d.ctx, key).Result()
	if err != nil {
		blog.Warnf("%s: remove from redis failed: %s", key, err)
	} else {
		blog.Infof("%s: remove from redis success", key)
	}
}

// GetInst get instance by objid,instkey and condition
func (d *Discover) GetInst(ownerID, objID string, instKey string, cond map[string]interface{}) (map[string]interface{},
	error) {
	rid := httpheader.GetRid(d.httpHeader)
	instData, err := d.GetInstFromRedis(instKey)
	if err == nil {
		blog.Infof("inst exist in redis: %s", instKey)
		return instData, nil
	} else {
		blog.Errorf("get inst from redis error: %s", err)
	}

	resp, err := d.CoreAPI.CoreService().Instance().ReadInstance(d.ctx, d.httpHeader, objID,
		&metadata.QueryCondition{Condition: cond})
	if err != nil {
		blog.Errorf("search inst failed, cond: %s, error: %s, rid: %s", cond, err.Error(), rid)
		return nil, fmt.Errorf("search inst failed: %s", err.Error())
	}

	if len(resp.Info) > 0 {
		val, err := json.Marshal(resp.Info[0])
		if err != nil {
			blog.Errorf("%s: flush to redis marshal failed: %s", instKey, err)
		}
		d.TrySetRedis(instKey, val, cacheTime)
		return resp.Info[0], nil
	}

	return nil, nil
}
