package metadata

import (
	"context"
	"fmt"
	"regexp"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/util"
	"configcenter/src/common/valid/attribute/manager/register"
)

func init() {
	// Register the array_longchar attribute type
	register.Register(arrayLongchar{})
}

type arrayLongchar struct {
}

// Name returns the name of the array_longchar attribute.
func (a arrayLongchar) Name() string {
	return "array_longchar"
}

// DisplayName returns the display name for user.
func (a arrayLongchar) DisplayName() string {
	return "长文本数组"
}

// RealType returns the db type of the array_longchar attribute.
// Flattened array uses LongChar as storage type
func (a arrayLongchar) RealType() string {
	return common.FieldTypeLongChar
}

// Info returns the tips for user.
func (a arrayLongchar) Info() string {
	return "长文本字符数组"
}

// Validate validates the array_longchar attribute value
func (a arrayLongchar) Validate(ctx context.Context, objID string, propertyType string, required bool, option interface{}, value interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	if value == nil {
		if required {
			blog.Errorf("array_longchar attribute %s.%s value is required but got nil, rid: %s", objID, propertyType, rid)
			return fmt.Errorf("array_longchar attribute %s.%s value is required but got nil", objID, propertyType)
		}
		return nil
	}

	// Validate that value is a slice of any
	strArray, ok := value.([]interface{})
	if !ok {
		blog.Errorf("array_longchar attribute %s.%s value must be []interface{}, got %T, rid: %s", objID, propertyType, value, rid)
		return fmt.Errorf("array_longchar attribute %s.%s value must be []interface{}, got %T", objID, propertyType, value)
	}

	// Parse option for regex pattern
	regex := common.FieldTypeLongCharRegexp

	arrayOpt, err := ParseArrayOption[any](option)
	if err != nil {
		return err
	}
	if arrayOpt.Option != nil {
		if optStr, ok := arrayOpt.Option.(string); ok && len(optStr) > 0 {
			regex = optStr
		}
	}

	// Compile regex pattern
	pattern, err := regexp.Compile(regex)
	if err != nil {
		blog.Errorf("array_longchar invalid regex pattern %s, err: %v, rid: %s", regex, err, rid)
		return fmt.Errorf("array_longchar invalid regex pattern: %v", err)
	}
	if arrayOpt.Cap > len(strArray) {
		return fmt.Errorf("array_longchar invalid cap %d, rid: %s", arrayOpt.Cap, rid)
	}
	// Validate each item in the array
	for i, item := range strArray {
		strVal, ok := item.(string)
		if !ok {
			blog.Errorf("array_longchar attribute %s.%s array item [%d] type %T is not string, rid: %s", objID, propertyType, i, item, rid)
			return fmt.Errorf("array_longchar attribute %s.%s array item [%d] type %T is not string", objID, propertyType, i, item)
		}

		// Validate length
		if len(strVal) > common.FieldTypeLongLenChar {
			blog.Errorf("array_longchar attribute %s.%s array item [%d] length %d exceeds max %d, rid: %s", objID, propertyType, i, len(strVal), common.FieldTypeLongLenChar, rid)
			return fmt.Errorf("array_longchar attribute %s.%s array item [%d] length exceeds max %d", objID, propertyType, i, common.FieldTypeLongLenChar)
		}

		// Validate regex
		if !pattern.MatchString(strVal) {
			blog.Errorf("array_longchar attribute %s.%s array item [%d] value does not match regex, rid: %s", objID, propertyType, i, rid)
			return fmt.Errorf("array_longchar attribute %s.%s array item [%d] does not match regex pattern", objID, propertyType, i)
		}
	}

	return nil
}

// FillLostValue fills the lost value with default value
func (a arrayLongchar) FillLostValue(ctx context.Context, valData mapstr.MapStr, propertyId string, defaultValue, option interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	valData[propertyId] = nil
	if defaultValue == nil {
		return nil
	}

	// Validate default value
	defaultArray, ok := defaultValue.([]interface{})
	if !ok {
		blog.Errorf("array_longchar default value must be []interface{}, got %T, rid: %s", defaultValue, rid)
		return fmt.Errorf("array_longchar default value must be []interface{}, got %T", defaultValue)
	}

	// Parse option for regex pattern
	regex := common.FieldTypeLongCharRegexp

	arrayOpt, err := ParseArrayOption[any](option)
	if err != nil {
		return err
	}
	if arrayOpt.Option != nil {
		if optStr, ok := arrayOpt.Option.(string); ok && len(optStr) > 0 {
			regex = optStr
		}
	}

	// Compile regex pattern
	pattern, err := regexp.Compile(regex)
	if err != nil {
		blog.Errorf("array_longchar invalid regex pattern %s, err: %v, rid: %s", regex, err, rid)
		return fmt.Errorf("array_longchar invalid regex pattern: %v", err)
	}

	// Validate each item in default array
	for i, item := range defaultArray {
		strVal, ok := item.(string)
		if !ok {
			blog.Errorf("array_longchar default value array item [%d] type %T is not string, rid: %s", i, item, rid)
			return fmt.Errorf("array_longchar default value array item [%d] type %T is not string", i, item)
		}

		if len(strVal) > common.FieldTypeLongLenChar {
			blog.Errorf("array_longchar default value array item [%d] length %d exceeds max %d, rid: %s", i, len(strVal), common.FieldTypeLongLenChar, rid)
			return fmt.Errorf("array_longchar default value array item [%d] length exceeds max %d", i, common.FieldTypeLongLenChar)
		}

		if !pattern.MatchString(strVal) {
			blog.Errorf("array_longchar default value array item [%d] does not match regex, rid: %s", i, rid)
			return fmt.Errorf("array_longchar default value array item [%d] does not match regex pattern", i)
		}
	}

	valData[propertyId] = defaultArray
	return nil
}

// ValidateOption validates the option field
func (a arrayLongchar) ValidateOption(ctx context.Context, option interface{}, defaultVal interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	// Validate regex pattern if provided
	arrayOpt, err := ParseArrayOption[any](option)
	if err != nil {
		return err
	}
	if optStr, ok := arrayOpt.Option.(string); ok && len(optStr) > 0 {
		if _, err := regexp.Compile(optStr); err != nil {
			blog.Errorf("array_longchar invalid regex pattern %s, err: %v, rid: %s", optStr, err, rid)
			return fmt.Errorf("array_longchar invalid regex pattern: %v", err)
		}
	}

	if defaultVal == nil {
		return nil
	}

	// Validate default value
	defaultArray, ok := defaultVal.([]interface{})
	if !ok {
		blog.Errorf("array_longchar default value must be []interface{}, got %T, rid: %s", defaultVal, rid)
		return fmt.Errorf("array_longchar default value must be []interface{}, got %T", defaultVal)
	}

	// Get regex pattern
	regex := common.FieldTypeLongCharRegexp
	if option != nil {
		if optStr, ok := option.(string); ok && len(optStr) > 0 {
			regex = optStr
		}
	}

	pattern, err := regexp.Compile(regex)
	if err != nil {
		blog.Errorf("array_longchar invalid regex pattern %s, err: %v, rid: %s", regex, err, rid)
		return fmt.Errorf("array_longchar invalid regex pattern: %v", err)
	}

	// Validate each item in default array
	for i, item := range defaultArray {
		strVal, ok := item.(string)
		if !ok {
			blog.Errorf("array_longchar default value array item [%d] type %T is not string, rid: %s", i, item, rid)
			return fmt.Errorf("array_longchar default value array item [%d] type %T is not string", i, item)
		}

		if len(strVal) > common.FieldTypeLongLenChar {
			blog.Errorf("array_longchar default value array item [%d] length exceeds max %d, rid: %s", i, common.FieldTypeLongLenChar, rid)
			return fmt.Errorf("array_longchar default value array item [%d] length exceeds max %d", i, common.FieldTypeLongLenChar)
		}

		if !pattern.MatchString(strVal) {
			blog.Errorf("array_longchar default value array item [%d] does not match regex, rid: %s", i, rid)
			return fmt.Errorf("array_longchar default value array item [%d] does not match regex pattern", i)
		}
	}

	return nil
}

var _ register.AttributeTypeI = &arrayLongchar{}
