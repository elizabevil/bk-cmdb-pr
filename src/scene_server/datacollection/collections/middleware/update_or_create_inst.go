package middleware

import (
	"fmt"

	"configcenter/src/common"
	"configcenter/src/common/auditlog"
	"configcenter/src/common/blog"
	httpheader "configcenter/src/common/http/header"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

// UpdateOrCreateInst update instance if existed, or create it if non-exist
func (d *Discover) UpdateOrCreateInst(msg *string) error {
	if msg == nil {
		return fmt.Errorf("message nil")
	}

	rid := httpheader.GetRid(d.httpHeader)
	ownerID := d.parseOwnerId(msg)
	objID := d.parseObjID(msg)

	// 解析消息数据
	bodyData, err := d.parseData(msg)
	if err != nil {
		return fmt.Errorf("parse data error: %s", err)
	}

	// 获取唯一键信息
	uniqueKeys, err := d.getUniqueKeys(rid, objID, ownerID)
	if err != nil {
		return fmt.Errorf("get unique keys failed: %s", err)
	}

	// 构建查询条件和实例键
	cond, instKeyStr, err := d.buildInstanceCondition(objID, ownerID, uniqueKeys, bodyData)
	if err != nil {
		return err
	}

	// 查找现有实例
	inst, err := d.GetInst(ownerID, objID, instKeyStr, cond)
	if err != nil {
		return fmt.Errorf("get inst error: %s", err)
	}

	blog.Infof("get inst result: %v", inst)

	// 根据实例是否存在决定创建或更新
	if len(inst) <= 0 {
		return d.createInstance(objID, bodyData, rid)
	}

	return d.updateInstance(objID, rid, instKeyStr, inst, bodyData)
}

// getUniqueKeys 获取模型的唯一键
func (d *Discover) getUniqueKeys(rid, objID string, ownerID string) ([]string, error) {
	// 获取必须检查的唯一键
	cond := map[string]any{
		common.BKObjIDField: objID,
		"must_check":        true,
	}

	uniqueResp, err := d.CoreAPI.CoreService().Model().ReadModelAttrUnique(d.ctx, d.httpHeader,
		metadata.QueryCondition{Condition: cond})
	if err != nil {
		blog.Errorf("search model unique failed, cond: %s, error: %s, rid: %s", cond, err.Error(), rid)
		return nil, err
	}

	if uniqueResp.Count != 1 {
		return nil, fmt.Errorf("model %s has wrong must check unique num", objID)
	}

	// 获取唯一键对应的属性ID
	keyIDs := make([]int64, 0, len(uniqueResp.Info[0].Keys))
	for _, key := range uniqueResp.Info[0].Keys {
		keyIDs = append(keyIDs, int64(key.ID))
	}

	// 查询属性详细信息
	attrCond := map[string]any{
		common.BKObjIDField:   objID,
		common.BKOwnerIDField: ownerID,
		common.BKFieldID: map[string]any{
			common.BKDBIN: keyIDs,
		},
	}

	attrResp, err := d.CoreAPI.CoreService().Model().ReadModelAttr(d.ctx, d.httpHeader, objID,
		&metadata.QueryCondition{Condition: attrCond})
	if err != nil {
		blog.Errorf("search model attribute failed, cond: %s, error: %s, rid: %s", attrCond, err.Error(), rid)
		return nil, err
	}

	if attrResp.Count <= 0 {
		blog.Errorf("unique model attribute count illegal, cond: %s, rid: %s", attrCond, rid)
		return nil, fmt.Errorf("search model attribute failed, return is empty")
	}

	keys := make([]string, 0, len(attrResp.Info))
	for _, attr := range attrResp.Info {
		keys = append(keys, attr.PropertyID)
	}

	return keys, nil
}

// buildInstanceCondition 构建实例查询条件
func (d *Discover) buildInstanceCondition(objID, ownerID string, keys []string,
	bodyData map[string]any) (map[string]any, string, error) {
	cond := map[string]any{
		common.BKObjIDField:   objID,
		common.BKOwnerIDField: ownerID,
	}

	valArr := make([]string, 0, len(keys)>>1)
	for _, key := range keys {
		val := util.GetStrByInterface(bodyData[key])
		if val == "" {
			return nil, "", fmt.Errorf("skip inst because of empty unique key %s value", key)
		}
		valArr = append(valArr, val)
		cond[key] = bodyData[key]
	}

	instKeyStr := d.CreateInstKey(objID, ownerID, valArr)
	return cond, instKeyStr, nil
}

// createInstance 创建新实例
func (d *Discover) createInstance(objID string, bodyData map[string]any, rid string) error {
	resp, err := d.CoreAPI.CoreService().Instance().CreateInstance(d.ctx, d.httpHeader, objID,
		&metadata.CreateModelInstance{Data: bodyData})
	if err != nil {
		blog.Errorf("create instance failed %s", err.Error())
		return fmt.Errorf("create instance failed: %s", err.Error())
	}

	blog.Infof("create inst result: %v", resp)

	// 记录审计日志
	return d.saveCreateAuditLog(objID, rid, bodyData)
}

// updateInstance 更新现有实例
func (d *Discover) updateInstance(objID, rid, instKeyStr string, inst map[string]any, bodyData map[string]any) error {
	instIDField := common.GetInstIDField(objID)

	instID, err := util.GetInt64ByInterface(inst[instIDField])
	if err != nil {
		return fmt.Errorf("get bk_inst_id failed: %s %s", inst[instIDField], err.Error())
	}

	// 检测数据变化
	dataChange, hasDiff := d.detectDataChanges(inst, bodyData)
	if !hasDiff {
		blog.Infof("no need to update inst")
		return nil
	}

	// 生成更新前的审计日志
	auditLog, err := d.generateUpdateAuditLog(objID, rid, instID, instIDField, inst)
	if err != nil {
		return err
	}

	// 执行更新
	err = d.performUpdate(objID, instID, instIDField, dataChange)
	if err != nil {
		return err
	}

	// 清除Redis缓存
	d.TryUnsetRedis(instKeyStr)

	// 保存审计日志
	return d.saveAuditLog(auditLog, objID, rid)
}

// detectDataChanges 检测数据变化
func (d *Discover) detectDataChanges(inst map[string]any, bodyData map[string]any) (map[string]any, bool) {
	dataChange := make(map[string]any)
	hasDiff := false

	for attrId, attrValue := range bodyData {
		if attrId == defaultRelateAttr {
			// 处理关联属性的特殊逻辑
			if d.handleRelationAttr(attrId, attrValue, inst, dataChange) {
				hasDiff = true
			}
			continue
		}

		if inst[attrId] != attrValue {
			dataChange[attrId] = attrValue
			blog.Debug("[changed] %s: %v ---> %v", attrId, inst[attrId], dataChange[attrId])
			hasDiff = true
		}
	}

	return dataChange, hasDiff
}

// handleRelationAttr 处理关联属性
func (d *Discover) handleRelationAttr(attrId string, attrValue any, inst map[string]any, dataChange map[string]any,
) bool {
	relateList, ok := inst[defaultRelateAttr].([]any)
	if !ok || len(relateList) != 1 {
		blog.Errorf("parse relation data failed, skip update: \n%v\n", inst[defaultRelateAttr])
		return false
	}

	relateObj, ok := relateList[0].(map[string]any)
	if !ok {
		blog.Errorf("parse relation object failed, skip update")
		return false
	}

	// 如果关联对象已存在，跳过更新
	if relateObj["id"] != "" && relateObj["id"] != "0" && relateObj["id"] != nil {
		blog.Infof("skip updating single relation attr: [%s]=%v, since it is existed:%v.",
			defaultRelateAttr, attrValue, relateObj["id"])
		return false
	}

	// 更新关联属性
	if val, ok := attrValue.(string); ok && val != "" {
		dataChange[defaultRelateAttr] = val
		blog.Debug("[relation changed] %s: %v ---> %v", defaultRelateAttr, "nil", dataChange[attrId])
		return true
	}

	return false
}

// saveCreateAuditLog 保存创建实例的审计日志
func (d *Discover) saveCreateAuditLog(objID, rid string, bodyData map[string]any) error {
	kit := d.buildAuditKit(rid)
	audit := auditlog.NewInstanceAudit(d.CoreAPI.CoreService())

	// 生成创建审计日志
	data := []mapstr.MapStr{mapstr.NewFromMap(bodyData)}
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit,
		metadata.AuditCreate).WithOperateFrom(metadata.FromDataCollection)

	auditLog, err := audit.GenerateAuditLog(generateAuditParameter, objID, data)
	if err != nil {
		blog.Errorf("generate instance audit log failed after create instance, objID: %s, err: %v, rid: %s",
			objID, err, rid)
		return err
	}

	// 保存审计日志
	if err := audit.SaveAuditLog(kit, auditLog...); err != nil {
		blog.Errorf("save instance audit log failed after create instance, objID: %s, err: %v, rid: %s",
			objID, err, rid)
		return err
	}

	return nil
}

// generateUpdateAuditLog 生成更新审计日志
func (d *Discover) generateUpdateAuditLog(objID string, rid string, instID int64, instIDField string,
	inst map[string]any) ([]metadata.AuditLog, error) {

	kit := d.buildAuditKit(rid)
	audit := auditlog.NewInstanceAudit(d.CoreAPI.CoreService())

	// 清理不可变字段
	cleanInst := d.cleanUnchangeableFields(inst, instIDField)

	// generate audit log before update instance.
	auditCond := map[string]any{instIDField: instID}
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditUpdate).
		WithOperateFrom(metadata.FromDataCollection).WithUpdateFields(cleanInst)

	auditLog, err := audit.GenerateAuditLogByCondGetData(generateAuditParameter, objID, auditCond)
	if err != nil {
		blog.Errorf("generate instance audit log failed before update instance, objID: %s, err: %v, rid: %s",
			objID, err, rid)
		return nil, err
	}

	return auditLog, nil
}

// cleanUnchangeableFields 清理不可变字段
func (d *Discover) cleanUnchangeableFields(inst map[string]any, instIDField string) map[string]any {
	cleanInst := make(map[string]any)
	for k, v := range inst {
		cleanInst[k] = v
	}

	// 删除不可变字段
	delete(cleanInst, common.BKObjIDField)
	delete(cleanInst, common.BKOwnerIDField)
	delete(cleanInst, common.BKDefaultField)
	delete(cleanInst, instIDField)
	delete(cleanInst, common.LastTimeField)
	delete(cleanInst, common.CreateTimeField)

	return cleanInst
}

// performUpdate 执行更新操作
func (d *Discover) performUpdate(objID string, instID int64, instIDField string, dataChange map[string]any) error {
	input := metadata.UpdateOption{
		Data: dataChange,
		Condition: map[string]any{
			instIDField: instID,
		},
		CanEditAll: true,
	}

	resp, err := d.CoreAPI.CoreService().Instance().UpdateInstance(d.ctx, d.httpHeader, objID, &input)
	if err != nil {
		blog.Errorf("update instance failed %s", err.Error())
		return fmt.Errorf("update instance failed: %s", err.Error())
	}

	blog.Infof("update inst result: %v", resp)
	return nil
}

// saveAuditLog 保存审计日志
func (d *Discover) saveAuditLog(auditLog []metadata.AuditLog, objID, rid string) error {
	kit := d.buildAuditKit(rid)
	audit := auditlog.NewInstanceAudit(d.CoreAPI.CoreService())

	if err := audit.SaveAuditLog(kit, auditLog...); err != nil {
		blog.Errorf("save instance audit log failed after update instance, objID: %s, err: %v, rid: %s",
			objID, err, rid)
		return err
	}

	return nil
}

// buildAuditKit 构建审计工具包
func (d *Discover) buildAuditKit(rid string) *rest.Kit {
	return &rest.Kit{
		Rid:             rid,
		Header:          d.httpHeader,
		Ctx:             d.ctx,
		CCError:         d.CCErr.CreateDefaultCCErrorIf(httpheader.GetLanguage(d.httpHeader)),
		User:            common.CCSystemCollectorUserName,
		SupplierAccount: common.BKDefaultOwnerID,
	}
}
