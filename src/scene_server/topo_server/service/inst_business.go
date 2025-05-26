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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"configcenter/src/ac"
	"configcenter/src/ac/iam"
	"configcenter/src/ac/meta"
	"configcenter/src/common"
	"configcenter/src/common/auth"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/mapstruct"
	"configcenter/src/common/metadata"
	gparams "configcenter/src/common/paraparse"
	"configcenter/src/common/querybuilder"
	"configcenter/src/common/util"
	"configcenter/src/thirdparty/hooks"
)

// CreateBusiness create a new business
func (s *Service) CreateBusiness(ctx *rest.Contexts) {
	data := make(mapstr.MapStr)
	if err := ctx.DecodeInto(&data); err != nil {
		ctx.RespAutoError(err)
		return
	}

	// authorize
	authRes := meta.ResourceAttribute{Basic: meta.Basic{Type: meta.Business, Action: meta.Create}}
	if resp, authorized := s.AuthManager.Authorize(ctx.Kit, authRes); !authorized {
		ctx.RespNoAuth(resp)
		return
	}

	if err := hooks.ValidateCreateBusinessHook(ctx.Kit, s.Engine.CoreAPI, data); err != nil {
		blog.Errorf("validate create business hook failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	data.Set(common.BKDefaultField, common.DefaultFlagDefaultValue)
	// do with transaction
	var business mapstr.MapStr
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		business, err = s.Logics.BusinessOperation().CreateBusiness(ctx.Kit, data)
		if err != nil {
			blog.Errorf("create business failed, err: %v, rid: %s", err, ctx.Kit.Rid)
			return err
		}

		// register business resource creator action to iam
		if auth.EnableAuthorize() {
			var bizID int64
			if bizID, err = business.Int64(common.BKAppIDField); err != nil {
				blog.Errorf("get biz id failed, err: %v, biz: %#v, rid: %s", err, business, ctx.Kit.Rid)
				return err
			}

			var bizName string
			if bizName, err = business.String(common.BKAppNameField); err != nil {
				blog.Errorf("get biz name failed, err: %v, biz: %#v, rid: %s", err, business, ctx.Kit.Rid)
				return err
			}

			iamInstance := metadata.IamInstanceWithCreator{
				Type:    string(iam.Business),
				ID:      strconv.FormatInt(bizID, 10),
				Name:    bizName,
				Creator: ctx.Kit.User,
			}
			_, err = s.AuthManager.Authorizer.RegisterResourceCreatorAction(ctx.Kit.Ctx, ctx.Kit.Header, iamInstance)
			if err != nil {
				blog.Errorf("register created business to iam failed, err: %v, rid: %s", err, ctx.Kit.Rid)
				return err
			}
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(business)
}

// UpdateBusiness update the business
func (s *Service) UpdateBusiness(ctx *rest.Contexts) {
	data := make(mapstr.MapStr)
	if err := ctx.DecodeInto(&data); err != nil {
		ctx.RespAutoError(err)
		return
	}

	bizID, err := strconv.ParseInt(ctx.Request.PathParameter("app_id"), 10, 64)
	if err != nil {
		blog.Errorf("failed to parse the biz id, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	cond := mapstr.MapStr{
		common.BKAppIDField: bizID,
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		err = s.Logics.InstOperation().UpdateInst(ctx.Kit, cond, data, common.BKInnerObjIDApp)
		if err != nil {
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

// UpdateBusinessStatus update the business status
func (s *Service) UpdateBusinessStatus(ctx *rest.Contexts) {
	data := struct {
		metadata.UpdateBusinessStatusOption `json:",inline"`
	}{}
	if err := ctx.DecodeInto(&data); err != nil {
		ctx.RespAutoError(err)
		return
	}

	bizID, err := strconv.ParseInt(ctx.Request.PathParameter("app_id"), 10, 64)
	if err != nil {
		blog.Errorf("failed to parse the biz id, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	query := &metadata.QueryCondition{
		Condition: mapstr.MapStr{common.BKAppIDField: bizID},
	}
	_, bizs, err := s.Logics.BusinessOperation().FindBiz(ctx.Kit, query)
	if len(bizs) <= 0 {
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommNotFound))
		return
	}
	bizName, err := bizs[0].String(common.BKAppNameField)
	if err != nil {
		blog.Errorf("parse biz failed, biz: %+v, err: %v, rid: %s", bizs[0], err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	updateData := make(mapstr.MapStr)
	switch common.DataStatusFlag(ctx.Request.PathParameter("flag")) {
	case common.DataStatusDisabled:
		err := s.Logics.InstAssociationOperation().CheckAssociations(ctx.Kit, common.BKInnerObjIDApp, []int64{bizID})
		if err != nil {
			ctx.RespAutoError(err)
			return
		}

		err = s.Logics.BusinessOperation().CheckArchiveBizResource(ctx.Kit, bizID)
		if err != nil {
			ctx.RespAutoError(err)
			return
		}

		achieveBizName, err := s.Logics.BusinessOperation().GenerateAchieveBusinessName(ctx.Kit, bizName)
		if err != nil {
			ctx.RespAutoError(err)
			return
		}
		updateData.Set(common.BKAppNameField, achieveBizName)
		updateData.Set(common.BKDataStatusField, ctx.Request.PathParameter("flag"))
	case common.DataStatusEnable:
		if len(data.UpdateBusinessStatusOption.BizName) > 0 {
			updateData.Set(common.BKAppNameField, data.UpdateBusinessStatusOption.BizName)
		}
		updateData.Set(common.BKDataStatusField, ctx.Request.PathParameter("flag"))
	default:
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, ctx.Request.PathParameter))
		return
	}

	cond := mapstr.MapStr{
		common.BKAppIDField: bizID,
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		err = s.Logics.InstOperation().UpdateInst(ctx.Kit, cond, updateData, common.BKInnerObjIDApp)
		if err != nil {
			blog.Errorf("UpdateBusinessStatus failed, run update failed, err: %+v, rid: %s", err, ctx.Kit.Rid)
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

// UpdateBizPropertyBatch batch update business properties
func (s *Service) UpdateBizPropertyBatch(ctx *rest.Contexts) {
	param := new(metadata.UpdateBizPropertyBatchParameter)
	if err := ctx.DecodeInto(param); err != nil {
		ctx.RespAutoError(err)
		return
	}

	bizIDs, err := s.getBizIDByCond(ctx, param.Condition)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	if len(bizIDs) <= 0 {
		blog.Errorf("found no business by condition, rid: %s", ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommNotFound))
		return
	}

	// check authorization
	if s.AuthManager.Enabled() {
		if err := s.AuthManager.AuthorizeByBusinessID(ctx.Kit.Ctx, ctx.Kit.Header, meta.Update, bizIDs...); err ==
			ac.NoAuthorizeError {
			perm, err := s.AuthManager.GenBizBatchNoPermissionResp(ctx.Kit.Ctx, ctx.Kit.Header, meta.Update, bizIDs)
			if err != nil && err != ac.NoAuthorizeError {
				blog.Errorf("get biz permission failed, biz: %v, err: %v, rid: %s", bizIDs, err, ctx.Kit.Rid)
				ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommAuthorizeFailed))
				return
			}
			ctx.RespEntityWithError(perm, ac.NoAuthorizeError)
			return
		} else if err != nil {
			blog.Errorf("biz authorize failed, biz: %v, err: %v, rid: %s", bizIDs, err, ctx.Kit.Rid)
			ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommAuthorizeFailed))
			return
		}
	}

	updateCond := mapstr.MapStr{
		common.BKAppIDField: mapstr.MapStr{
			common.BKDBIN: bizIDs,
		},
		// exclude archived biz
		common.BKDataStatusField: mapstr.MapStr{
			common.BKDBNE: common.DataStatusDisabled,
		},
		// exclude default biz
		common.BKDefaultField: mapstr.MapStr{
			common.BKDBNE: common.DefaultAppFlag,
		},
	}

	data := param.Properties
	// cannot update biz bk_data_status
	delete(data, common.BKDataStatusField)

	// update biz instances
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		err = s.Logics.InstOperation().UpdateInst(ctx.Kit, updateCond, data, common.BKInnerObjIDApp)
		if err != nil {
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

func (s *Service) getBizIDByCond(ctx *rest.Contexts, cond mapstr.MapStr) ([]int64, error) {
	opt := &metadata.QueryCondition{
		Condition: mapstr.MapStr{
			metadata.AttributeFieldObjectID:     common.BKInnerObjIDApp,
			metadata.AttributeFieldPropertyType: common.FieldTypeUser,
		},
		DisableCounter: true,
	}
	resp, err := s.Engine.CoreAPI.CoreService().Model().ReadModelAttr(ctx.Kit.Ctx, ctx.Kit.Header,
		common.BKInnerObjIDApp, opt)
	if err != nil {
		blog.Errorf("failed to get the business attribute, err: %v, rid:%s", err, ctx.Kit.Rid)
		return nil, err
	}
	// userFieldArr Fields in the business are user-type fields
	var userFields []string
	for _, attribute := range resp.Info {
		userFields = append(userFields, attribute.PropertyID)
	}

	filter := handleSpecialBusinessFieldSearchCond(cond, userFields)
	// can only find normal business, but not resource pool business
	filter[common.BKDefaultField] = mapstr.MapStr{
		common.BKDBNE: common.DefaultAppFlag,
	}

	distinctOpt := &metadata.DistinctFieldOption{
		TableName: common.BKTableNameBaseApp,
		Field:     common.BKAppIDField,
		Filter:    filter,
	}

	rst, err := s.Engine.CoreAPI.CoreService().Common().GetDistinctField(ctx.Kit.Ctx, ctx.Kit.Header, distinctOpt)
	if err != nil {
		blog.Errorf("get biz ids failed, distinct opt: %+v, err: %v, rid: %s", distinctOpt, err, ctx.Kit.Rid)
		return nil, err
	}

	bizIDs, err := util.SliceInterfaceToInt64(rst)
	if err != nil {
		blog.Errorf("biz ids to int failed, biz ids: %v, err: %v, rid: %s", rst, err, ctx.Kit.Rid)
		return nil, err
	}

	return bizIDs, nil
}

// DeleteBusiness delete archived business
func (s *Service) DeleteBusiness(ctx *rest.Contexts) {
	param := new(metadata.DeleteBizParam)
	if err := ctx.DecodeInto(param); err != nil {
		ctx.RespAutoError(err)
		return
	}

	if len(param.BizID) == 0 {
		blog.Errorf("invalid bk_biz_id len 0, rid: %s", ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, common.BKAppIDField))
		return
	}

	if len(param.BizID) > common.BKDefaultLimit {
		blog.Errorf("bk_biz_id len %d exceed max page size %d, rid:%s", len(param.BizID),
			common.BKDefaultLimit, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommXXExceedLimit, "update",
			common.BKDefaultLimit))
		return
	}

	if err := hooks.ValidateDeleteBusinessHook(ctx.Kit, s.Engine.CoreAPI, param.BizID); err != nil {
		blog.Errorf("validate delete business hook failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	// delete biz instances and related resources
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		if err := s.Logics.BusinessOperation().DeleteBusiness(ctx.Kit, param.BizID); err != nil {
			blog.Errorf("failed to delete biz, ids: %v, err: %v, rid: %s", param.BizID, err, ctx.Kit.Rid)
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

// SearchReducedBusinessList TODO
// find business list with these info：
// 1. have any authorized resources in a business.
// 2. only returned with a few field for this business info.
func (s *Service) SearchReducedBusinessList(ctx *rest.Contexts) {
	page := metadata.BasePage{
		Limit: common.BKNoLimit,
	}
	sortParam := ctx.Request.QueryParameter("sort")
	if len(sortParam) > 0 {
		page.Sort = sortParam
	}

	if errKey, err := page.Validate(true); err != nil {
		blog.Errorf("page parameter invalid, errKey: %v, err: %s, rid: %s", errKey, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, errKey))
		return
	}

	query := &metadata.QueryCondition{
		Fields: []string{common.BKAppIDField, common.BKAppNameField},
		Page:   page,
		Condition: mapstr.MapStr{
			common.BKDataStatusField: mapstr.MapStr{common.BKDBNE: common.DataStatusDisabled},
			common.BKDefaultField:    0,
		},
	}

	if s.AuthManager.Enabled() {
		authInput := meta.ListAuthorizedResourcesParam{
			UserName:     ctx.Kit.User,
			ResourceType: meta.Business,
			Action:       meta.ViewBusinessResource,
		}
		authorizedRes, err := s.AuthManager.Authorizer.ListAuthorizedResources(ctx.Kit.Ctx, ctx.Kit.Header, authInput)
		if err != nil {
			blog.Errorf("[api-biz] SearchReducedBusinessList failed, ListAuthorizedResources failed, user: %s,"+
				" err: %s, rid: %s", ctx.Kit.User, err.Error(), ctx.Kit.Rid)
			ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrorTopoGetAuthorizedBusinessListFailed))
			return
		}

		// if isAny is false,we should add bizIds condition.
		if !authorizedRes.IsAny {
			appList := make([]int64, 0)
			for _, resourceID := range authorizedRes.Ids {
				bizID, err := strconv.ParseInt(resourceID, 10, 64)
				if err != nil {
					blog.Errorf("parse bizID(%s) failed, err: %v, rid: %s", bizID, err, ctx.Kit.Rid)
					ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsNeedInt, common.BKAppIDField))
					return
				}
				appList = append(appList, bizID)
			}
			if len(appList) == 0 {
				ctx.RespEntityWithCount(0, make([]mapstr.MapStr, 0))
				return
			}
			// sort for prepare to find business with page.
			sort.Sort(util.Int64Slice(appList))
			// user can only find business that is already authorized.
			query.Condition[common.BKAppIDField] = mapstr.MapStr{common.BKDBIN: appList}
		}
	}

	cnt, instItems, err := s.Logics.BusinessOperation().FindBiz(ctx.Kit, query)
	if nil != err {
		blog.Errorf("failed to find the objects(%s), error info is %s, rid: %s",
			ctx.Request.PathParameter("obj_id"), err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	datas := make([]mapstr.MapStr, 0)
	for _, item := range instItems {
		inst := mapstr.New()
		inst[common.BKAppIDField] = item[common.BKAppIDField]
		inst[common.BKAppNameField] = item[common.BKAppNameField]
		datas = append(datas, inst)
	}

	result := mapstr.MapStr{
		"count": cnt,
		"info":  datas,
	}
	ctx.RespEntity(result)
}

// GetBusinessBasicInfo search biz basic info
func (s *Service) GetBusinessBasicInfo(ctx *rest.Contexts) {
	bizID, err := strconv.ParseInt(ctx.Request.PathParameter("app_id"), 10, 64)
	if err != nil {
		blog.Errorf("failed to parse the biz id, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	query := &metadata.QueryCondition{
		Fields: []string{common.BKAppNameField, common.BKAppIDField},
		Condition: map[string]interface{}{
			common.BKAppIDField: bizID,
		},
	}
	result, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header,
		common.BKInnerObjIDApp, query)
	if err != nil {
		blog.Errorf("get business failed, bizID: %s, err: %v, rid: %s", bizID, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if len(result.Info) == 0 {
		blog.Errorf("get business by id not found, bizID: %d, rid: %s", bizID, ctx.Kit.Rid)
		err := ctx.Kit.CCError.CCError(common.CCErrCommNotFound)
		ctx.RespAutoError(err)
		return
	}

	bizData := result.Info[0]
	ctx.RespEntity(bizData)
}

// 4 scenarios, such as user's name user1, scenarios as follows:
// user1
// user1,user3
// user2,user1
// user2,user1,user4
const exactUserRegexp = `(^USER_PLACEHOLDER$)|(^USER_PLACEHOLDER[,]{1})|([,]{1}USER_PLACEHOLDER[,]{1})` +
	`|([,]{1}USER_PLACEHOLDER$)`

func handleSpecialBusinessFieldSearchCond(input map[string]interface{}, userFieldArr []string) map[string]interface{} {
	output := make(map[string]interface{})
	exactAnd := make([]map[string]interface{}, 0)
	for i, j := range input {
		if j == nil {
			output[i] = j
			continue
		}

		objType := reflect.TypeOf(j)
		switch objType.Kind() {
		case reflect.String:
			if _, ok := j.(json.Number); ok {
				output[i] = j
				continue
			}
			targetStr := j.(string)
			if util.InStrArr(userFieldArr, i) {
				for _, user := range strings.Split(strings.Trim(targetStr, ","), ",") {
					// search with exactly the user's name with regexpF
					like := strings.Replace(exactUserRegexp, "USER_PLACEHOLDER", gparams.SpecialCharChange(user), -1)
					exactAnd = append(exactAnd, mapstr.MapStr{i: mapstr.MapStr{common.BKDBLIKE: like}})
				}
			} else {
				attrVal := gparams.SpecialCharChange(targetStr)
				output[i] = map[string]interface{}{common.BKDBLIKE: attrVal, common.BKDBOPTIONS: "i"}
			}
		default:
			output[i] = j
		}
	}

	if len(exactAnd) > 0 {
		output[common.BKDBAND] = exactAnd
	}

	return output
}

func (s *Service) SearchBusiness(ctx *rest.Contexts) {
	searchCond := new(metadata.QueryBusinessRequest)
	if err := ParseParamsFromCtx(ctx, &searchCond); err != nil {
		return
	}

	if searchCond.Condition != nil && searchCond.BizPropertyFilter != nil {
		blog.Errorf("condition and biz_property_filter cannot both be set, rid: %s", ctx.Kit.Rid)
		ctx.RespErrorCodeOnly(common.CCErrCommParamsInvalid, "condition and biz_property_filter cannot be set at the same time")
		return
	}
	if err := s.injectUserTypeFields(ctx, searchCond); err != nil {
		ctx.RespAutoError(err)
		return
	}

	bizIDs, err := s.extractBizIDs(searchCond.Condition)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsInvalid, "", common.BKAppIDField)
		return
	}

	if s.AuthManager.Enabled() {
		if err := s.filterByAuthorization(ctx, searchCond, bizIDs); err != nil {
			return
		}
	}

	if err := s.applyBizPropertyFilter(ctx, searchCond); err != nil {
		return
	}

	s.ensureDefaultCondition(searchCond)

	query := &metadata.QueryCondition{
		Condition:     searchCond.Condition,
		TimeCondition: searchCond.TimeCondition,
		Fields:        searchCond.Fields,
		Page:          searchCond.Page,
	}

	count, items, err := s.Logics.BusinessOperation().FindBiz(ctx.Kit, query)
	if err != nil {
		blog.Errorf("find business failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	result := mapstr.MapStr{
		"count": count,
		"info":  items,
	}
	ctx.RespEntity(result)
}
func (s *Service) injectUserTypeFields(ctx *rest.Contexts, req *metadata.QueryBusinessRequest) error {
	opt := &metadata.QueryCondition{
		Condition: mapstr.MapStr{
			metadata.AttributeFieldObjectID:     common.BKInnerObjIDApp,
			metadata.AttributeFieldPropertyType: common.FieldTypeUser,
		},
		DisableCounter: true,
	}
	attrArr, err := s.Engine.CoreAPI.CoreService().Model().ReadModelAttr(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDApp, opt)
	if err != nil {
		blog.Errorf("failed to get business attribute, err: %s, rid: %s", err.Error(), ctx.Kit.Rid)
		return err
	}

	userFields := make([]string, 0, len(attrArr.Info))
	for _, attr := range attrArr.Info {
		userFields = append(userFields, attr.PropertyID)
	}
	req.Condition = handleSpecialBusinessFieldSearchCond(req.Condition, userFields)
	return nil
}

// 提取bk_biz_id
func (s *Service) extractBizIDs(cond mapstr.MapStr) ([]int64, error) {
	var ids []int64
	val, exists := cond[common.BKAppIDField]
	if !exists {
		return ids, nil
	}

	switch v := val.(type) {
	case map[string]interface{}:
		if eq, ok := v[common.BKDBEQ]; ok {
			id, err := util.GetInt64ByInterface(eq)
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		if in, ok := v[common.BKDBIN]; ok {
			for _, raw := range in.([]interface{}) {
				id, err := util.GetInt64ByInterface(raw)
				if err != nil {
					return nil, err
				}
				ids = append(ids, id)
			}
		}
	default:
		id, err := util.GetInt64ByInterface(val)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// filterByAuthorization
func (s *Service) filterByAuthorization(ctx *rest.Contexts, req *metadata.QueryBusinessRequest, bizIDs []int64) error {
	authInput := meta.ListAuthorizedResourcesParam{
		UserName:     ctx.Kit.User,
		ResourceType: meta.Business,
		Action:       meta.Find,
	}
	authRes, err := s.AuthManager.Authorizer.ListAuthorizedResources(ctx.Kit.Ctx, ctx.Kit.Header, authInput)
	if err != nil {
		blog.Errorf("ListAuthorizedResources failed, user: %s, err: %v, rid: %s", ctx.Kit.User, err, ctx.Kit.Rid)
		ctx.RespErrorCodeOnly(common.CCErrorTopoGetAuthorizedBusinessListFailed, "")
		return err
	}

	if authRes.IsAny {
		if len(bizIDs) > 0 {
			req.Condition[common.BKAppIDField] = mapstr.MapStr{common.BKDBIN: bizIDs}
		}
		return nil
	}

	appList := make([]int64, 0, len(authRes.Ids))
	for _, id := range authRes.Ids {
		bid, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsNeedInt, common.BKAppIDField))
			return err
		}
		appList = append(appList, bid)
	}

	if len(bizIDs) > 0 {
		authBizIDs := []int64{}
		for _, id := range bizIDs {
			if util.InArray(id, appList) {
				authBizIDs = append(authBizIDs, id)
			}
		}
		if len(authBizIDs) == 0 {
			ctx.RespEntityWithCount(0, []mapstr.MapStr{})
			return errors.New("no authorized biz")
		}
		req.Condition[common.BKAppIDField] = mapstr.MapStr{common.BKDBIN: authBizIDs}
		return nil
	}

	if len(appList) == 0 {
		ctx.RespEntityWithCount(0, []mapstr.MapStr{})
		return errors.New("no authorized biz")
	}

	sort.Sort(util.Int64Slice(appList))
	req.Condition[common.BKAppIDField] = mapstr.MapStr{common.BKDBIN: appList}
	return nil
}

// applyBizPropertyFilter
func (s *Service) applyBizPropertyFilter(ctx *rest.Contexts, req *metadata.QueryBusinessRequest) error {
	if req.BizPropertyFilter == nil {
		return nil
	}

	option := &querybuilder.RuleOption{
		NeedSameSliceElementType: true,
		MaxSliceElementsCount:    querybuilder.DefaultMaxSliceElementsCount,
		MaxConditionOrRulesCount: querybuilder.DefaultMaxConditionOrRulesCount,
	}

	if key, err := req.BizPropertyFilter.Validate(option); err != nil {
		blog.Errorf("invalid BizPropertyFilter, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, fmt.Sprintf("biz.property.%s", key)))
		return err
	}

	if req.BizPropertyFilter.GetDeep() > querybuilder.MaxDeep {
		err := fmt.Errorf("exceed max query condition depth: %d", querybuilder.MaxDeep)
		blog.Errorf("bizPropertyFilter too deep, rid: %s", ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, err.Error()))
		return err
	}

	mgoFilter, key, err := req.BizPropertyFilter.ToMgo()
	if err != nil {
		blog.Errorf("ToMgo failed: %s, err: %v, rid: %s", req.BizPropertyFilter, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, err.Error()+fmt.Sprintf(", biz_property_filter.%s", key)))
		return err
	}
	req.Condition = mgoFilter
	return nil
}

// ensureDefaultCondition
func (s *Service) ensureDefaultCondition(req *metadata.QueryBusinessRequest) {
	if _, ok := req.Condition[common.BKDataStatusField]; !ok {
		req.Condition[common.BKDataStatusField] = mapstr.MapStr{common.BKDBNE: common.DataStatusDisabled}
	}
	req.Condition[common.BKDefaultField] = 0
}

// ParseParamsFromCtx
func ParseParamsFromCtx[T any](ctx *rest.Contexts, req *T) error {
	if err := ctx.DecodeInto(req); err != nil {
		blog.Errorf("failed to parse the params, error info is %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespErrorCodeOnly(common.CCErrCommJSONUnmarshalFailed, "")
		return err
	}
	return nil
}

// GetResourcePoolBiz get resource pool business info, right now only returns biz id
func (s *Service) GetResourcePoolBiz(ctx *rest.Contexts) {
	// authorize using list resource pool hosts permission
	authRes := meta.ResourceAttribute{Basic: meta.Basic{Type: meta.HostInstance, Action: meta.Find}}
	if resp, authorized := s.AuthManager.Authorize(ctx.Kit, authRes); !authorized {
		ctx.RespNoAuth(resp)
		return
	}

	query := &metadata.QueryCondition{
		Condition: mapstr.MapStr{
			common.BKDefaultField: common.DefaultAppFlag,
		},
		Fields: []string{common.BKAppIDField},
		Page: metadata.BasePage{
			Limit: 1,
		},
	}

	_, bizInfo, err := s.Logics.BusinessOperation().FindBiz(ctx.Kit, query)
	if err != nil {
		blog.Errorf("get %s resource pool biz id failed, err: %v, rid: %s", ctx.Kit.TenantID, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if len(bizInfo) == 0 {
		blog.Errorf("%s resource pool biz not found, rid: %s", ctx.Kit.TenantID, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, "resource pool business not found"))
		return
	}

	ctx.RespEntity(bizInfo[0])
}

// ListAllBusinessSimplify list all businesses with return only several fields
func (s *Service) ListAllBusinessSimplify(ctx *rest.Contexts) {
	page := metadata.BasePage{
		Limit: common.BKNoLimit,
	}
	sortParam := ctx.Request.QueryParameter("sort")
	if len(sortParam) > 0 {
		page.Sort = sortParam
	}

	fields := []string{
		common.BKAppIDField,
		common.BKAppNameField,
	}

	query := &metadata.QueryCondition{
		Fields: fields,
		Page:   page,
		Condition: mapstr.MapStr{
			common.BKDataStatusField: mapstr.MapStr{common.BKDBNE: common.DataStatusDisabled},
		},
	}
	cnt, instItems, err := s.Logics.BusinessOperation().FindBiz(ctx.Kit, query)
	if err != nil {
		blog.Errorf("find business failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	businesses := make([]metadata.BizBasicInfo, 0)
	for _, item := range instItems {
		business := metadata.BizBasicInfo{}
		if err := mapstruct.Decode2Struct(item, &business); err != nil {
			blog.Errorf("decode biz from db failed, err: %v, rid: %s", err, ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}
		businesses = append(businesses, business)
	}

	result := map[string]interface{}{
		"count": cnt,
		"info":  businesses,
	}
	ctx.RespEntity(result)
}

// GetBriefTopologyNodeRelation is used to get directly related business topology node information.
// As is, you can find modules belongs to a set; or you can find the set a module belongs to.
func (s *Service) GetBriefTopologyNodeRelation(ctx *rest.Contexts) {
	options := new(metadata.GetBriefBizRelationOptions)
	if err := ctx.DecodeInto(options); err != nil {
		ctx.RespAutoError(err)
		return
	}

	rawErr := options.Validate()
	if rawErr.ErrCode != 0 {
		blog.Errorf("validate failed, err: %v, rid: %s", rawErr.Args, ctx.Kit.Rid)
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	relations, err := s.Logics.BusinessOperation().GetBriefTopologyNodeRelation(ctx.Kit, options)
	if err != nil {
		blog.Errorf("get brief topology node relation failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(&relations)
}
