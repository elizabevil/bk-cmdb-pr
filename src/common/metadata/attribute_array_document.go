package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/util"
	"configcenter/src/common/valid/attribute/manager/register"
)

func init() {
	register.Register(&arrayDocument{})
}

// arrayDocument represents a document array attribute type.
type arrayDocument struct{}

func (a *arrayDocument) Name() string {
	return "array_document"
}

func (a *arrayDocument) DisplayName() string {
	return "文件数组"
}

func (a *arrayDocument) RealType() string {
	return common.FieldTypeLongChar
}

func (a *arrayDocument) Info() string {
	return "附件数组"
}

// Validate validates the arrayDocument attribute value.
func (a *arrayDocument) Validate(ctx context.Context, objID, propertyType string,
	required bool, option, value interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)
	if value == nil {
		if required {
			blog.Errorf("array_document %s.%s required, rid: %s", objID, propertyType, rid)
			return fmt.Errorf("array_document %s.%s required", objID, propertyType)
		}
		return nil
	}

	arr, ok := value.([]interface{})
	if !ok {
		blog.Errorf("array_document %s.%s not []interface{}, rid: %s",
			objID, propertyType, rid)
		return fmt.Errorf("array_document %s.%s must be array", objID, propertyType)
	}

	opts, err := a.parseDocumentOption(option)
	if err != nil {
		blog.Errorf("array_document parse option failed: %v, rid: %s", err, rid)
		return fmt.Errorf("array_document invalid option: %v", err)
	}

	return a.validateDocArray(rid, objID, propertyType, arr, opts)
}

// FillLostValue fills missing values with default value.
func (a *arrayDocument) FillLostValue(ctx context.Context, valData mapstr.MapStr,
	propertyID string, defaultValue, option interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	valData[propertyID] = nil
	if defaultValue == nil {
		return nil
	}

	arr, ok := defaultValue.([]interface{})
	if !ok {
		blog.Errorf("array_document default not []interface{}, rid: %s", rid)
		return fmt.Errorf("array_document default must be array")
	}

	opts, err := a.parseDocumentOption(option)
	if err != nil {
		blog.Errorf("array_document parse option failed: %v, rid: %s", err, rid)
		return fmt.Errorf("array_document invalid option: %v", err)
	}

	if err := a.validateDocArray(rid, "", "", arr, opts); err != nil {
		return err
	}

	valData[propertyID] = arr
	return nil
}

// ValidateOption validates the option field.
func (a *arrayDocument) ValidateOption(ctx context.Context, option,
	defaultVal interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)
	opts, err := a.parseDocumentOption(option)
	if err != nil {
		blog.Errorf("array_document parse option failed: %v, rid: %s", err, rid)
		return fmt.Errorf("array_document invalid option: %v", err)
	}

	if err := a.validateDocOption(rid, opts); err != nil {
		return err
	}

	if defaultVal == nil {
		return nil
	}

	arr, ok := defaultVal.([]interface{})
	if !ok {
		blog.Errorf("array_document default not []interface{}, rid: %s", rid)
		return fmt.Errorf("array_document default must be array")
	}

	return a.validateDocArray(rid, "", "", arr, opts)
}

// validateDocOption validates document option settings.
func (a *arrayDocument) validateDocOption(rid string,
	opts ArrayOption[documentOption]) error {

	if len(opts.Option.AllowSuffixes) == 0 {
		blog.Errorf("array_document allow_suffixes required, rid: %s", rid)
		return fmt.Errorf("array_document allow_suffixes required")
	}
	if len(opts.Option.Type) == 0 {
		blog.Errorf("array_document type required, rid: %s", rid)
		return fmt.Errorf("array_document type required")
	}

	validTypes := map[string]struct{}{
		"image": {}, "document": {}, "video": {}, "audio": {},
	}
	if _, ok := validTypes[opts.Option.Type]; !ok {
		blog.Errorf("array_document type %s unsupported, rid: %s", opts.Option.Type, rid)
		return fmt.Errorf("array_document type %s unsupported", opts.Option.Type)
	}

	if len(opts.Option.Regex) > 0 {
		if _, err := regexp.Compile(opts.Option.Regex); err != nil {
			blog.Errorf("array_document regex %s invalid: %v, rid: %s",
				opts.Option.Regex, err, rid)
			return fmt.Errorf("array_document regex invalid: %v", err)
		}
	}
	return nil
}

// validateDocArray validates all documents in array.
func (a *arrayDocument) validateDocArray(rid, objID, prop string,
	arr []interface{}, opts ArrayOption[documentOption]) error {

	if opts.Cap > len(arr) {
		return fmt.Errorf("array_float invalid cap %d, rid: %s", opts.Cap, rid)
	}
	for i, v := range arr {
		doc, err := a.parseDocumentValue(v)
		if err != nil {
			if objID != "" {
				blog.Errorf("array_document %s.%s item [%d] invalid, rid: %s",
					objID, prop, i, rid)
				return fmt.Errorf("array_document %s.%s item [%d] invalid", objID, prop, i)
			}
			return fmt.Errorf("array_document item [%d] invalid", i)
		}

		if len(doc.Name) > common.FieldTypeLongLenChar ||
			len(doc.Value) > common.FieldTypeLongLenChar {
			if objID != "" {
				blog.Errorf("array_document %s.%s item [%d] length exceeded, rid: %s",
					objID, prop, i, rid)
				return fmt.Errorf("array_document %s.%s item [%d] length exceeded",
					objID, prop, i)
			}
			return fmt.Errorf("array_document item [%d] length exceeded", i)
		}

		if len(opts.Option.Regex) > 0 {
			match, err := regexp.MatchString(opts.Option.Regex, doc.Value)
			if err != nil || !match {
				if objID != "" {
					blog.Errorf("array_document %s.%s item [%d] regex unmatched, rid: %s",
						objID, prop, i, rid)
					return fmt.Errorf("array_document %s.%s item [%d] regex unmatched",
						objID, prop, i)
				}
				return fmt.Errorf("array_document item [%d] regex unmatched", i)
			}
		}
	}
	return nil
}

// parseDocumentOption parses the option into documentOption.
func (a *arrayDocument) parseDocumentOption(option interface{}) (
	ArrayOption[documentOption], error) {

	arrayOption, err := ParseArrayOption[documentOption](option)
	if err != nil {
		return arrayOption, err
	}
	switch val := option.(type) {
	case *documentOption:
		arrayOption.Option = *val
		return arrayOption, nil
	case documentOption:
		arrayOption.Option = val
		return arrayOption, nil
	}
	optBytes, err := json.Marshal(option)
	if err != nil {
		return arrayOption,
			fmt.Errorf("array_document invalid option type: %v", option)
	}
	res := documentOption{}
	if err := json.Unmarshal(optBytes, &res); err != nil {
		return arrayOption,
			fmt.Errorf("array_document invalid option: %v, err: %v", option, err)
	}
	arrayOption.Option = res
	return arrayOption, nil
}

// parseDocumentValue parses the value into valueDocument.
func (a *arrayDocument) parseDocumentValue(value interface{}) (valueDocument, error) {

	if value == nil {
		return valueDocument{}, nil
	}
	switch val := value.(type) {
	case *valueDocument:
		return *val, nil
	case valueDocument:
		return val, nil
	}

	valBytes, err := json.Marshal(value)
	if err != nil {
		return valueDocument{},
			fmt.Errorf("array_document invalid value type: %v", value)
	}
	res := valueDocument{}
	if err := json.Unmarshal(valBytes, &res); err != nil {
		return valueDocument{},
			fmt.Errorf("array_document invalid value: %v, err: %v", value, err)
	}

	return res, nil
}

var _ register.AttributeTypeI = (*arrayDocument)(nil)
