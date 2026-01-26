package metadata

import (
	"context"
	"fmt"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/util"
	"configcenter/src/common/valid/attribute/manager/register"
)

func init() {
	// Register the arrayBool attribute type
	register.Register(arrayBool{})
}

type arrayBool struct {
}

// Name returns the name of the arrayBool attribute.
func (a arrayBool) Name() string {
	return "array_bool"
}

// DisplayName returns the display name for user.
func (a arrayBool) DisplayName() string {
	return "布尔数组"
}

// RealType returns the db type of the arrayBool attribute.
// Flattened array uses LongChar as storage type
func (a arrayBool) RealType() string {
	return common.FieldTypeLongChar
}

// Info returns the tips for user.
func (a arrayBool) Info() string {
	return "布尔值的扁平化数组字段，存储多个true/false值"
}

// Validate validates the arrayBool attribute value
func (a arrayBool) Validate(ctx context.Context, objID string, propertyType string, required bool, option interface{}, value interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	if value == nil {
		if required {
			blog.Errorf("arrayBool attribute %s.%s value is required but got nil, rid: %s", objID, propertyType, rid)
			return fmt.Errorf("arrayBool attribute %s.%s value is required but got nil", objID, propertyType)
		}
		return nil
	}
	opts, err := ParseArrayOption[any](option)
	if err != nil {
		blog.Errorf("array_bool parse option failed: %v, rid: %s", err, rid)
		return fmt.Errorf("array_bool invalid option: %v", err)
	}

	// Validate that value is a slice of any
	boolArray, ok := value.([]interface{})
	if !ok {
		blog.Errorf("arrayBool attribute %s.%s value must be []interface{}, got %T, rid: %s", objID, propertyType, value, rid)
		return fmt.Errorf("arrayBool attribute %s.%s value must be []interface{}, got %T", objID, propertyType, value)
	}
	if opts.Cap > len(boolArray) {
		return fmt.Errorf("array_bool invalid cap %d, rid: %s", opts.Cap, rid)
	}
	// Validate each item in the array is a boolean
	for i, item := range boolArray {
		if _, ok := item.(bool); !ok {
			blog.Errorf("arrayBool attribute %s.%s array item [%d] type %T is not bool, rid: %s", objID, propertyType, i, item, rid)
			return fmt.Errorf("arrayBool attribute %s.%s array item [%v] type %T is not bool", objID, propertyType, item, item)
		}
	}

	return nil
}

// FillLostValue fills the lost value with default value
func (a arrayBool) FillLostValue(ctx context.Context, valData mapstr.MapStr, propertyId string, defaultValue, option interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	valData[propertyId] = nil
	if defaultValue == nil {
		return nil
	}

	// Validate default value
	defaultArray, ok := defaultValue.([]interface{})
	if !ok {
		blog.Errorf("arrayBool default value must be []interface{}, got %T, rid: %s", defaultValue, rid)
		return fmt.Errorf("arrayBool default value must be []interface{}, got %T", defaultValue)
	}

	// Validate each item in default array
	for i, item := range defaultArray {
		if _, ok := item.(bool); !ok {
			blog.Errorf("arrayBool default value array item [%d] type %T is not bool, rid: %s", i, item, rid)
			return fmt.Errorf("arrayBool default value array item [%d] type %T is not bool", i, item)
		}
	}

	valData[propertyId] = defaultArray
	return nil
}

// ValidateOption validates the option field
func (a arrayBool) ValidateOption(ctx context.Context, option interface{}, defaultVal interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	_, err := ParseArrayOption[any](option)
	if err != nil {
		return err
	}
	if defaultVal == nil {
		return nil
	}

	// Validate default value
	defaultArray, ok := defaultVal.([]interface{})
	if !ok {
		blog.Errorf("arrayBool default value must be []interface{}, got %T, rid: %s", defaultVal, rid)
		return fmt.Errorf("arrayBool default value must be []interface{}, got %T", defaultVal)
	}

	// Validate each item in default array
	for i, item := range defaultArray {
		if _, ok := item.(bool); !ok {
			blog.Errorf("arrayBool default value array item [%d] type %T is not bool, rid: %s", i, item, rid)
			return fmt.Errorf("arrayBool default value array item [%d] type %T is not bool", i, item)
		}
	}

	return nil
}

var _ register.AttributeTypeI = &arrayBool{}
