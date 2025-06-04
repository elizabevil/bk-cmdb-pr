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

package v3v0v8

import (
	"context"

	"configcenter/src/common"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/types"

	"go.mongodb.org/mongo-driver/bson"
	"gopkg.in/mgo.v2"
)

const (
	tableNameSubscription = "cc_Subscription"
	subscriptionIDField   = "subscription_id"
)

func createTable(ctx context.Context, db dal.RDB, conf *upgrader.Config) (err error) {
	for tableName, indexes := range tables {
		exists, err := db.HasTable(ctx, tableName)
		if err != nil {
			return err
		}
		if !exists {
			if err = db.CreateTable(ctx, tableName); err != nil && !mgo.IsDup(err) {
				return err
			}
		}
		for index := range indexes {
			if err = db.Table(tableName).CreateIndex(ctx, indexes[index]); err != nil && !db.IsDuplicatedError(err) {
				return err
			}
		}
	}
	return nil
}

var tables = map[string][]types.Index{
	common.BKTableNameBaseApp: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAppIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAppNameField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKDefaultField, Value: 1}}, Background: true},
	},

	common.BKTableNameBaseHost: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKHostIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKHostNameField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKHostInnerIPField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKHostOuterIPField, Value: 1}}, Background: true},
	},
	common.BKTableNameBaseModule: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKModuleIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKModuleNameField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKDefaultField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAppIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKSetIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKParentIDField, Value: 1}}, Background: true},
	},
	common.BKTableNameModuleHostConfig: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAppIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKHostIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKModuleIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKSetIDField, Value: 1}}, Background: true},
	},
	common.BKTableNameObjAsst: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKObjIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAsstObjIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
	},
	common.BKTableNameObjAttDes: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKObjIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKFieldID, Value: 1}}, Background: true},
	},
	common.BKTableNameObjClassification: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKClassificationIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKClassificationNameField, Value: 1}}, Background: true},
	},
	common.BKTableNameObjDes: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKObjIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKClassificationIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKObjNameField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
	},
	common.BKTableNameBaseInst: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKObjIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKInstIDField, Value: 1}}, Background: true},
	},
	common.BKTableNameAuditLog: {
		{Name: "index_bk_supplier_account", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
		{Name: "index_audit_type", Keys: bson.D{{Key: common.BKAuditTypeField, Value: 1}}, Background: true},
		{Name: "index_action", Keys: bson.D{{Key: common.BKActionField, Value: 1}}, Background: true},
	},
	common.BKTableNameBasePlat: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
	},
	"cc_Proc2Module": {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAppIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKProcIDField, Value: 1}}, Background: true},
	},
	common.BKTableNameBaseProcess: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKProcIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAppIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
	},
	common.BKTableNamePropertyGroup: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKObjIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKPropertyGroupIDField, Value: 1}}, Background: true},
	},
	common.BKTableNameBaseSet: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKSetIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKParentIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKAppIDField, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BkSupplierAccount, Value: 1}}, Background: true},
		types.Index{Name: "", Keys: bson.D{{Key: common.BKSetNameField, Value: 1}}, Background: true},
	},
	tableNameSubscription: {
		types.Index{Name: "", Keys: bson.D{{Key: subscriptionIDField, Value: 1}}, Background: true},
	},
	common.BKTableNameTopoGraphics: {
		types.Index{Name: "", Keys: bson.D{{Key: "scope_type", Value: 1}, {Key: "scope_id", Value: 1}, {Key: "node_type", Value: 1},
			{Key: common.BKObjIDField, Value: 1}, {Key: common.BKInstIDField, Value: 1}}, Background: true, Unique: true},
	},
	common.BKTableNameInstAsst: {
		types.Index{Name: "", Keys: bson.D{{Key: common.BKObjIDField, Value: 1}, {Key: common.BKInstIDField, Value: 1}},
			Background: true},
	},

	common.BKTableNameHistory:      {},
	common.BKTableNameHostFavorite: {},
	common.BKTableNameUserAPI:      {},
	common.BKTableNameUserCustom:   {},
	common.BKTableNameIDgenerator:  {},
	common.BKTableNameSystem:       {},
	common.BKTableNameDelArchive:   {},
}
