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
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/mapstruct"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// 解码与校验过程
func (ps *ProcServer) decodeAndValidateProcesses(kit *rest.Kit, input *metadata.UpdateRawProcessInstanceInput) ([]int64, errors.CCErrorCoder) {
	processIDs := make([]int64, 0, len(input.Raw))
	input.Processes = make([]metadata.Process, 0, len(input.Raw))
	var process metadata.Process
	for _, pData := range input.Raw {
		if err := mapstr.DecodeFromMapStr(&process, pData); err != nil {
			blog.Errorf("unmarshal process failed, data: %+v, err: %v, rid: %s", pData, err, kit.Rid)
			return nil, kit.CCError.CCError(common.CCErrCommJSONUnmarshalFailed)
		}
		if process.ProcessID == 0 {
			return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcessIDField)
		}
		input.Processes = append(input.Processes, process)
		processIDs = append(processIDs, process.ProcessID)
	}
	return util.IntArrayUnique(processIDs), nil
}

// 获取进程关系
func (ps *ProcServer) listProcessRelations(kit *rest.Kit, bizID int64, processIDs []int64) (*metadata.MultipleProcessInstanceRelation, errors.CCErrorCoder) {
	option := &metadata.ListProcessInstanceRelationOption{
		BusinessID: bizID,
		ProcessIDs: processIDs,
		Page:       metadata.BasePage{Limit: common.BKNoLimit},
	}
	relations, err := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(kit.Ctx, kit.Header, option)
	if err != nil {
		blog.Errorf("list process relation failed, err: %v, rid: %s", err, kit.Rid)
		return nil, err
	}
	return relations, nil
}

// 校验 ProcessID 是否存在
func (ps *ProcServer) validateProcessesExist(kit *rest.Kit, processIDs []int64, relations []metadata.ProcessInstanceRelation) errors.CCErrorCoder {
	foundProcessIDs := make(map[int64]bool)
	for _, r := range relations {
		foundProcessIDs[r.ProcessID] = true
	}
	invalid := make([]string, 0, len(processIDs)>>3)
	for _, id := range processIDs {
		if !foundProcessIDs[id] {
			invalid = append(invalid, strconv.FormatInt(id, 10))
		}
	}
	if len(invalid) > 0 {
		blog.Errorf("invalid process IDs: %v, rid: %s", invalid, kit.Rid)
		msg := fmt.Sprintf("[%s: %s]", common.BKProcessIDField, strings.Join(invalid, ","))
		return kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, msg)
	}
	return nil
}

// 提取 hostID
func (ps *ProcServer) extractHostIDs(relations []metadata.ProcessInstanceRelation) []int64 {
	hostIDs := make([]int64, 0, len(relations)>>3)
	for _, rel := range relations {
		if rel.ProcessTemplateID != common.ServiceTemplateIDNotSet {
			hostIDs = append(hostIDs, rel.HostID)
		}
	}
	return hostIDs
}

// 获取模板 Map
func (ps *ProcServer) getProcessTemplates(kit *rest.Kit, relations []metadata.ProcessInstanceRelation) (map[int64]*metadata.ProcessTemplate, errors.CCErrorCoder) {
	processTemplateMap := make(map[int64]*metadata.ProcessTemplate)
	for _, relation := range relations {
		if relation.ProcessTemplateID == common.ServiceTemplateIDNotSet {
			continue
		}
		if _, ok := processTemplateMap[relation.ProcessTemplateID]; ok {
			continue
		}
		template, err := ps.CoreAPI.CoreService().Process().GetProcessTemplate(kit.Ctx, kit.Header, relation.ProcessTemplateID)
		if err != nil {
			blog.Errorf("get process template failed, ID: %d, err: %v, rid: %s", relation.ProcessTemplateID, err, kit.Rid)
			return nil, err
		}
		processTemplateMap[relation.ProcessTemplateID] = template
	}
	return processTemplateMap, nil
}

// 构造更新数据
func (ps *ProcServer) buildProcessUpdateData(
	kit *rest.Kit,
	input metadata.UpdateRawProcessInstanceInput,
	relations []metadata.ProcessInstanceRelation,
	processTemplateMap map[int64]*metadata.ProcessTemplate,
	hostMap map[int64]map[string]any,
) (map[int64]map[string]any, errors.CCErrorCoder) {

	process2ServiceInstanceMap := make(map[int64]metadata.ProcessInstanceRelation)
	for _, r := range relations {
		process2ServiceInstanceMap[r.ProcessID] = r
	}

	processDataMap := make(map[int64]map[string]any)
	for idx, process := range input.Processes {
		relation, exist := process2ServiceInstanceMap[process.ProcessID]
		if !exist {
			err := kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcessIDField)
			blog.Errorf("process related service instance not found, process: %+v, err: %v, rid: %s", process, err, kit.Rid)
			return nil, err
		}

		raw := input.Raw[idx]
		clearFields := metadata.FilterValidFields(getNilFields(raw))

		var (
			processData map[string]any
			err         error
		)

		if relation.ProcessTemplateID == common.ServiceTemplateIDNotSet {
			process.BusinessID = input.BizID
			processData, err = mapstruct.Struct2Map(process)
			if err != nil {
				blog.Errorf("json Unmarshal process failed, processData: %+v, err: %v, rid: %s", processData, err, kit.Rid)
				return nil, kit.CCError.CCError(common.CCErrCommJsonDecode)
			}
			for _, field := range []string{common.BKProcessIDField, common.MetadataField, common.LastTimeField, common.CreateTimeField} {
				delete(processData, field)
			}
		} else {
			processTemplate, exist := processTemplateMap[relation.ProcessTemplateID]
			if !exist {
				err := kit.CCError.CCError(common.CCErrCommNotFound)
				blog.Errorf("process related processTemplate not found, relation: %+v, err: %v, rid: %s", relation, err, kit.Rid)
				return nil, err
			}
			processData, err = processTemplate.ExtractInstanceUpdateData(&process, hostMap[relation.HostID])
			if err != nil {
				blog.Errorf("process related processTemplate not found, relation: %+v, err: %v, rid: %s", relation, err, kit.Rid)
				return nil, errors.New(common.CCErrCommParamsInvalid, err.Error())
			}
			clearFields = processTemplate.GetEditableFields(clearFields)
		}

		for _, field := range clearFields {
			processData[field] = nil
		}
		processDataMap[process.ProcessID] = processData
	}
	return processDataMap, nil
}

func getNilFields(raw map[string]any) []string {
	var fields []string
	for k, v := range raw {
		if v == nil {
			fields = append(fields, k)
		}
	}
	return fields
}

// 并发批量更新
func (ps *ProcServer) batchUpdateProcessInstances(kit *rest.Kit, dataMap map[int64]map[string]any) errors.CCErrorCoder {
	var wg sync.WaitGroup
	var firstErr atomic.Value
	pipeline := make(chan struct{}, 10)

	for pid, data := range dataMap {
		wg.Add(1)
		pipeline <- struct{}{}
		go func(processID int64, processData map[string]any) {
			defer func() {
				<-pipeline
				wg.Done()
			}()
			err := ps.Logic.UpdateProcessInstance(kit, processID, processData)
			if err != nil {
				blog.Errorf("UpdateProcessInstance failed, ID: %d, data: %+v, err: %v, rid: %s", processID, processData, err, kit.Rid)
				firstErr.CompareAndSwap(nil, err)
			}
		}(pid, data)
	}
	wg.Wait()

	if val := firstErr.Load(); val != nil {
		return val.(errors.CCErrorCoder)
	}
	return nil
}

// updateProcessInstances
func (ps *ProcServer) updateProcessInstances(ctx *rest.Contexts, input metadata.UpdateRawProcessInstanceInput) ([]int64,
	errors.CCErrorCoder) {
	bizID := input.BizID

	processIDs, err := ps.decodeAndValidateProcesses(ctx.Kit, &input)
	if err != nil {
		return nil, err
	}

	relations, err := ps.listProcessRelations(ctx.Kit, bizID, processIDs)
	if err != nil {
		return nil, err
	}

	if err := ps.validateProcessesExist(ctx.Kit, processIDs, relations.Info); err != nil {
		return nil, err
	}

	hostIDs := ps.extractHostIDs(relations.Info)
	processTemplateMap, err := ps.getProcessTemplates(ctx.Kit, relations.Info)
	if err != nil {
		return nil, err
	}

	hostMap, err := ps.Logic.GetHostIPMapByID(ctx.Kit, hostIDs)
	if err != nil {
		return nil, err
	}

	processDataMap, err := ps.buildProcessUpdateData(ctx.Kit, input, relations.Info, processTemplateMap, hostMap)
	if err != nil {
		return nil, err
	}

	if err := ps.batchUpdateProcessInstances(ctx.Kit, processDataMap); err != nil {
		return nil, err
	}

	return processIDs, nil
}
