/*
 * Tencent is pleased to support the open source community by making
 * 蓝鲸智云 - 配置平台 (BlueKing - Configuration System) available.
 * Copyright (C) 2017 THL A29 Limited,
 * a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on
 * an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package logics

import (
	"context"
	"fmt"

	"configcenter/pkg/tenant"
	"configcenter/pkg/tenant/logics"
	"configcenter/src/apimachinery"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	commontypes "configcenter/src/common/types"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/mongo/local"
	"configcenter/src/storage/dal/mongo/sharding"
)

// NewTenantInterface get new tenant cli interface
type NewTenantInterface interface {
	NewTenantCli(tenant string) (local.DB, string, error)
}

// GetNewTenantCli get new tenant db
func GetNewTenantCli(kit *rest.Kit, cli interface{}) (local.DB, string, error) {
	newTenantCli, ok := cli.(NewTenantInterface)
	if !ok {
		blog.Errorf("get new tenant cli failed, rid: %s", kit.Rid)
		return nil, "", fmt.Errorf("get new tenant cli failed")
	}

	dbCli, dbUUID, err := newTenantCli.NewTenantCli(kit.TenantID)
	if err != nil || dbCli == nil {
		blog.Errorf("get new tenant cli failed, err: %v, tenant: %s, rid: %s", err, kit.TenantID, kit.Rid)
		return nil, "", fmt.Errorf("get new tenant cli failed, err: %v", err)
	}

	return dbCli, dbUUID, nil
}

// RefreshTenants refresh tenant info, skip tenant verify for apiserver
func RefreshTenants(coreAPI apimachinery.ClientSetInterface, db dal.Dal) error {
	tenants, err := logics.GetAllTenantsFromDB(context.Background(),
		db.Shard(sharding.NewShardOpts().WithIgnoreTenant()))
	if err != nil {
		blog.Errorf("get all tenants failed, err: %v", err)
		return err
	}
	tenant.SetTenant(tenants)

	needRefreshServer := []string{commontypes.CC_MODULE_APISERVER, commontypes.CC_MODULE_TASK,
		commontypes.CC_MODULE_CACHESERVICE, commontypes.CC_MODULE_EVENTSERVER, commontypes.CC_MODULE_SYNC}
	for _, module := range needRefreshServer {
		_, err = coreAPI.Refresh().RefreshTenant(module)
		if err != nil {
			blog.Errorf("refresh tenant info failed, module: %s, err: %v", module, err)
			return err
		}
	}

	return nil
}
