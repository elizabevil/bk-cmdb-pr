// Package service TODO
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
package service

import (
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"fmt"
	"strconv"
)

// ListProcessRelatedInfo list process related info according to condition
func (ps *ProcServer) ListProcessRelatedInfo(ctx *rest.Contexts) {
	bizID, err := parseBizID(ctx)
	if err != nil {
		return
	}

	input := new(metadata.ListProcessRelatedInfoOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}
	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}
	//=========
	moduleIDs, err := ps.getModuleIDsFromSet(ctx, bizID, input)
	if err != nil {
		return
	}

	serviceInstanceIDs, err := ps.getServiceInstanceIDs(ctx, bizID, moduleIDs, input)
	if err != nil {
		return
	}

	processIDs, err := ps.getProcessIDs(ctx, bizID, serviceInstanceIDs)
	if err != nil {
		return
	}

	finalFilter, fields, err := buildFinalFilter(ctx, bizID, processIDs, input)
	if err != nil {
		return
	}

	processResult, err := ps.fetchProcessDetails(ctx, finalFilter, fields, input.Page)
	if err != nil {
		return
	}

	if len(processResult.Info) == 0 {
		ctx.RespEntityWithCount(0, []interface{}{})
		return
	}

	processIDsNeed, processDetailMap := extractProcessDetails(processResult)

	ps.listProcessRelatedInfo(ctx, bizID, processIDsNeed, processDetailMap, int64(processResult.Count))
}

// parseBizID
func parseBizID(ctx *rest.Contexts) (int64, error) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		blog.Errorf("parse bk_biz_id error, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, "bk_biz_id"))
	}
	return bizID, err
}

//✂️ 示例：提取 getModuleIDsFromSet

func (ps *ProcServer) getModuleIDsFromSet(ctx *rest.Contexts,
	bizID int64,
	input *metadata.ListProcessRelatedInfoOption,
) ([]int64, error) {
	if len(input.Set.SetIDs) == 0 {
		return input.Module.ModuleIDs, nil
	}

	filter := map[string]interface{}{
		common.BKAppIDField: bizID,
		common.BKSetIDField: map[string]interface{}{common.BKDBIN: input.Set.SetIDs},
	}
	if len(input.Module.ModuleIDs) > 0 {
		filter[common.BKModuleIDField] = map[string]interface{}{common.BKDBIN: input.Module.ModuleIDs}
	}

	param := &metadata.QueryCondition{
		Condition: filter,
		Fields:    []string{common.BKModuleIDField},
		Page:      metadata.BasePage{Limit: common.BKNoLimit},
	}

	moduleResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, param)
	if err != nil {
		blog.Errorf("getModuleIDsFromSet error, param: %v, err: %v, rid: %s", param, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return nil, err
	}

	var mIDs []int64
	for _, info := range moduleResult.Info {
		if mID, err := info.Int64(common.BKModuleIDField); err == nil {
			mIDs = append(mIDs, mID)
		}
	}
	return mIDs, nil
}

func (ps *ProcServer) getServiceInstanceIDs(
	ctx *rest.Contexts,
	bizID int64,
	moduleIDs []int64,
	input *metadata.ListProcessRelatedInfoOption,
) ([]int64, error) {

	// 如果没有模块 ID 和服务实例 ID，直接返回空
	if len(input.ServiceInstance.IDs) == 0 && len(moduleIDs) == 0 {
		return nil, nil
	}

	// 构造过滤条件
	filter := map[string]interface{}{
		common.BKAppIDField: bizID,
	}

	if len(input.ServiceInstance.IDs) > 0 {
		filter[common.BKFieldID] = map[string]interface{}{
			common.BKDBIN: input.ServiceInstance.IDs,
		}
	}

	if len(moduleIDs) > 0 {
		filter[common.BKModuleIDField] = map[string]interface{}{
			common.BKDBIN: moduleIDs,
		}
	}

	option := &metadata.DistinctFieldOption{
		TableName: common.BKTableNameServiceInstance,
		Field:     common.BKFieldID,
		Filter:    filter,
	}

	sIDs, err := ps.CoreAPI.CoreService().Common().GetDistinctField(ctx.Kit.Ctx, ctx.Kit.Header, option)
	if err != nil {
		blog.Errorf("getServiceInstanceIDs: GetDistinctField failed, err: %v, option: %#v, rid: %s", err, *option, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return nil, err
	}

	// 转换为 int64
	srvInstIDs := make([]int64, len(sIDs))
	for idx, sID := range sIDs {
		if ID, err := strconv.ParseInt(fmt.Sprintf("%v", sID), 10, 64); err == nil {
			srvInstIDs[idx] = ID
		}
	}

	return srvInstIDs, nil
}

func (ps *ProcServer) getProcessIDs(
	ctx *rest.Contexts,
	bizID int64,
	serviceInstanceIDs []int64,
) ([]int64, error) {

	if len(serviceInstanceIDs) == 0 {
		return nil, nil
	}

	filter := map[string]interface{}{
		common.BKAppIDField: bizID,
		common.BKServiceInstanceIDField: map[string]interface{}{
			common.BKDBIN: serviceInstanceIDs,
		},
	}

	option := &metadata.DistinctFieldOption{
		TableName: common.BKTableNameProcessInstanceRelation,
		Field:     common.BKProcessIDField,
		Filter:    filter,
	}

	pIDs, err := ps.CoreAPI.CoreService().Common().GetDistinctField(ctx.Kit.Ctx, ctx.Kit.Header, option)
	if err != nil {
		blog.Errorf("getProcessIDs: GetDistinctField failed, err: %v, option: %#v, rid: %s", err, *option, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return nil, err
	}

	procIDs := make([]int64, len(pIDs))
	for idx, pID := range pIDs {
		if ID, err := strconv.ParseInt(fmt.Sprintf("%v", pID), 10, 64); err == nil {
			procIDs[idx] = ID
		}
	}

	return procIDs, nil
}

func buildFinalFilter(
	ctx *rest.Contexts,
	bizID int64,
	processIDs []int64,
	input *metadata.ListProcessRelatedInfoOption,
) (map[string]interface{}, []string, error) {

	// 基础过滤条件
	baseFilter := map[string]interface{}{
		common.BKAppIDField: bizID,
	}
	if len(processIDs) > 0 {
		baseFilter[common.BKProcessIDField] = map[string]interface{}{
			common.BKDBIN: processIDs,
		}
	}

	// 处理 propertyFilter
	propertyFilter := make(map[string]interface{})
	if input.ProcessPropertyFilter != nil {
		mgoFilter, key, err := input.ProcessPropertyFilter.ToMgo()
		if err != nil {
			errMsg := fmt.Sprintf("buildFinalFilter: ToMgo failed, err: %v, key: %s, rid: %s", err, key, ctx.Kit.Rid)
			blog.Errorf(errMsg)
			ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, err.Error()+", host_property_filter."+key))
			return nil, nil, err
		}
		if len(mgoFilter) > 0 {
			propertyFilter = mgoFilter
		}
	}

	// 合并 filter
	finalFilter := map[string]interface{}{}
	if len(propertyFilter) > 0 {
		finalFilter[common.BKDBAND] = []map[string]interface{}{baseFilter, propertyFilter}
	} else {
		finalFilter = baseFilter
	}

	// 处理字段
	fields := input.Fields
	if len(fields) > 0 {
		fields = append(fields, common.BKProcessIDField, common.BKProcessNameField, common.BKFuncIDField)
	}

	return finalFilter, fields, nil
}

func (ps *ProcServer) fetchProcessDetails(
	ctx *rest.Contexts,
	filter map[string]interface{},
	fields []string,
	page metadata.BasePage,
) (*metadata.InstDataInfo, error) {

	if page.Sort == "" {
		page.Sort = common.BKProcessIDField
	}

	query := &metadata.QueryCondition{
		Fields:    fields,
		Page:      page,
		Condition: filter,
	}

	result, err := ps.CoreAPI.CoreService().Instance().ReadInstance(
		ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDProc, query)
	if err != nil {
		blog.Errorf("fetchProcessDetails: ReadInstance failed, err: %v, query: %+v, rid: %s", err, query, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return nil, err
	}

	return result, nil
}

func extractProcessDetails(processResult *metadata.InstDataInfo) ([]int64, map[int64]interface{}) {
	processIDsNeed := make([]int64, len(processResult.Info))
	processDetailMap := map[int64]interface{}{}
	for idx, process := range processResult.Info {
		processID, _ := process.Int64(common.BKProcessIDField)
		processIDsNeed[idx] = processID
		processDetailMap[processID] = process
	}
	return processIDsNeed, processDetailMap
}
