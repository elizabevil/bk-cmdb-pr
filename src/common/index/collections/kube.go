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

package collections

import (
	"configcenter/src/common"
	kubetypes "configcenter/src/kube/types"
	"configcenter/src/storage/dal/types"

	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	registerIndexes(kubetypes.BKTableNameBaseCluster, commClusterIndexes)
	registerIndexes(kubetypes.BKTableNameBaseNode, commNodeIndexes)
	registerIndexes(kubetypes.BKTableNameBaseNamespace, commNamespaceIndexes)
	registerIndexes(kubetypes.BKTableNameBasePod, commPodIndexes)
	registerIndexes(kubetypes.BKTableNameBaseContainer, commContainerIndexes)
	registerIndexes(kubetypes.BKTableNameNsSharedClusterRel, nsSharedClusterRelIndexes)

	workLoadTables := []string{
		kubetypes.BKTableNameBaseDeployment, kubetypes.BKTableNameBaseDaemonSet,
		kubetypes.BKTableNameBaseStatefulSet, kubetypes.BKTableNameGameStatefulSet,
		kubetypes.BKTableNameGameDeployment, kubetypes.BKTableNameBaseCronJob,
		kubetypes.BKTableNameBaseJob, kubetypes.BKTableNameBasePodWorkload,
	}
	for _, table := range workLoadTables {
		registerIndexes(table, commWorkLoadIndexes)
	}
}

var commWorkLoadIndexes = []types.Index{
	{
		Name: common.CCLogicUniqueIdxNamePrefix + common.BKFieldID,
		Keys: bson.D{
			{Key: common.BKFieldID, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "bk_namespace_id_name",
		Keys: bson.D{
			{Key: kubetypes.BKNamespaceIDField, Value: 1},
			{Key: common.BKFieldName, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "cluster_uid",
		Keys: bson.D{
			{Key: kubetypes.ClusterUIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "cluster_id",
		Keys: bson.D{
			{Key: kubetypes.BKClusterIDFiled, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "name",
		Keys: bson.D{
			{Key: common.BKFieldName, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
}

var commContainerIndexes = []types.Index{
	{
		Name: common.CCLogicUniqueIdxNamePrefix + common.BKFieldID,
		Keys: bson.D{
			{Key: common.BKFieldID, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "bk_pod_id_container_uid",
		Keys: bson.D{
			{Key: kubetypes.BKPodIDField, Value: 1},
			{Key: kubetypes.ContainerUIDField, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "pod_id",
		Keys: bson.D{
			{Key: kubetypes.BKPodIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "biz_id",
		Keys: bson.D{
			{Key: kubetypes.BKBizIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "cluster_id",
		Keys: bson.D{
			{Key: kubetypes.BKClusterIDFiled, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "namespace_id",
		Keys: bson.D{
			{Key: kubetypes.BKNamespaceIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "reference_id_reference_kind",
		Keys: bson.D{
			{Key: kubetypes.RefIDField, Value: 1},
			{Key: kubetypes.RefKindField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
}

var commPodIndexes = []types.Index{
	{
		Name: common.CCLogicUniqueIdxNamePrefix + common.BKFieldID,
		Keys: bson.D{
			{Key: common.BKFieldID, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "bk_reference_id_reference_kind_name",
		Keys: bson.D{
			{Key: kubetypes.RefIDField, Value: 1},
			{Key: kubetypes.RefKindField, Value: 1},
			{Key: common.BKFieldName, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "reference_name_reference_kind",
		Keys: bson.D{
			{Key: kubetypes.RefNameField, Value: 1},
			{Key: kubetypes.RefIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "cluster_id",
		Keys: bson.D{
			{Key: kubetypes.BKClusterIDFiled, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "cluster_uid",
		Keys: bson.D{
			{Key: kubetypes.ClusterUIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "namespace_id",
		Keys: bson.D{
			{Key: kubetypes.BKNamespaceIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "reference_id_reference_kind",
		Keys: bson.D{
			{Key: kubetypes.RefIDField, Value: 1},
			{Key: kubetypes.RefKindField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "name",
		Keys: bson.D{
			{Key: common.BKFieldName, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "bk_host_id",
		Keys: bson.D{
			{Key: common.BKHostIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "node_id",
		Keys: bson.D{
			{Key: kubetypes.BKNodeIDField, Value: 1},
		},
		Background: true,
	},
}

var commNamespaceIndexes = []types.Index{
	{
		Name: common.CCLogicUniqueIdxNamePrefix + common.BKFieldID,
		Keys: bson.D{
			{Key: common.BKFieldID, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "bk_cluster_id_name",
		Keys: bson.D{
			{Key: kubetypes.BKClusterIDFiled, Value: 1},
			{Key: common.BKFieldName, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "cluster_uid",
		Keys: bson.D{
			{Key: kubetypes.ClusterUIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "cluster_id",
		Keys: bson.D{
			{Key: kubetypes.BKClusterIDFiled, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "name",
		Keys: bson.D{
			{Key: common.BKFieldName, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
}

var commNodeIndexes = []types.Index{
	{
		Name: common.CCLogicUniqueIdxNamePrefix + common.BKFieldID,
		Keys: bson.D{
			{Key: common.BKFieldID, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "bk_cluster_id_name",
		Keys: bson.D{
			{Key: kubetypes.BKClusterIDFiled, Value: 1},
			{Key: common.BKFieldName, Value: 1},
		},
		Unique:     true,
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "biz_id_cluster_uid",
		Keys: bson.D{
			{Key: common.BKAppIDField, Value: 1},
			{Key: kubetypes.ClusterUIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "biz_id_cluster_id",
		Keys: bson.D{
			{Key: common.BKAppIDField, Value: 1},
			{Key: kubetypes.BKClusterIDFiled, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "biz_id_host_id",
		Keys: bson.D{
			{Key: common.BKAppIDField, Value: 1},
			{Key: common.BKHostIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "biz_id_name",
		Keys: bson.D{
			{Key: common.BKAppIDField, Value: 1},
			{Key: common.BKFieldName, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
}

var commClusterIndexes = []types.Index{
	{
		Name: common.CCLogicUniqueIdxNamePrefix + common.BKFieldID,
		Keys: bson.D{
			{Key: common.BKFieldID, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "uid",
		Keys: bson.D{
			{Key: kubetypes.UidField, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "bk_biz_id_name",
		Keys: bson.D{
			{Key: common.BKAppIDField, Value: 1},
			{Key: common.BKFieldName, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + common.BKAppIDField,
		Keys: bson.D{
			{Key: common.BKAppIDField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "xid",
		Keys: bson.D{
			{Key: kubetypes.XidField, Value: 1},
			{Key: common.BkSupplierAccount, Value: 1},
		},
		Background: true,
	},
}

var nsSharedClusterRelIndexes = []types.Index{
	{
		Name: common.CCLogicUniqueIdxNamePrefix + "namespace_id",
		Keys: bson.D{
			{Key: kubetypes.BKNamespaceIDField, Value: 1},
		},
		Background: true,
		Unique:     true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "biz_id",
		Keys: bson.D{
			{Key: kubetypes.BKBizIDField, Value: 1},
		},
		Background: true,
	},
	{
		Name: common.CCLogicIndexNamePrefix + "asst_biz_id",
		Keys: bson.D{
			{Key: kubetypes.BKAsstBizIDField, Value: 1},
		},
		Background: true,
	},
}
