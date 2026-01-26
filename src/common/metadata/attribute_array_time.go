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
	// Register the array_time attribute type
	register.Register(arrayTime{})
}

type arrayTime struct {
}

// Name returns the name of the array_time attribute.
func (a arrayTime) Name() string {
	return "array_time"
}

// DisplayName returns the display name for user.
func (a arrayTime) DisplayName() string {
	return "时间数组"
}

// RealType returns the db type of the array_time attribute.
// Flattened array uses LongChar as storage type
func (a arrayTime) RealType() string {
	return common.FieldTypeLongChar
}

// Info returns the tips for user.
func (a arrayTime) Info() string {
	return "时间数组"
}

// Validate validates the array_time attribute value
func (a arrayTime) Validate(ctx context.Context, objID string, propertyType string, required bool,
	option interface{}, value interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	if value == nil {
		if required {
			blog.Errorf("array_time attribute %s.%s value is required but got nil, rid: %s", objID, propertyType, rid)
			return fmt.Errorf("array_time attribute %s.%s value is required but got nil", objID, propertyType)
		}
		return nil
	}
	arrayOpt, err := ParseArrayOption[any](option)
	if err != nil {
		return err
	}

	// Validate that value is a slice of any
	timeArray, ok := value.([]interface{})
	if !ok {
		blog.Errorf("array_time attribute %s.%s value must be []interface{}, got %T, rid: %s", objID, propertyType, value, rid)
		return fmt.Errorf("array_time attribute %s.%s value must be []interface{}, got %T", objID, propertyType, value)
	}
	if arrayOpt.Cap > len(timeArray) {
		return fmt.Errorf("array_time invalid cap %d, rid: %s", arrayOpt.Cap, rid)
	}
	// Validate each item in the array
	for i, item := range timeArray {
		// Validate time format
		if _, ok := util.IsTime(item); !ok {
			blog.Errorf("array_time attribute %s.%s array item [%d] type %T is not a valid time, rid: %s", objID, propertyType, i, item, rid)
			return fmt.Errorf("array_time attribute %s.%s array item [%d] is not a valid time", objID, propertyType, item)
		}
	}

	return nil
}

// FillLostValue fills the lost value with default value
func (a arrayTime) FillLostValue(ctx context.Context, valData mapstr.MapStr, propertyId string,
	defaultValue, option interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	valData[propertyId] = nil
	if defaultValue == nil {
		return nil
	}
	_, err := ParseArrayOption[any](option)
	if err != nil {
		return err
	}
	// Validate default value
	defaultArray, ok := defaultValue.([]interface{})
	if !ok {
		blog.Errorf("array_time default value must be []interface{}, got %T, rid: %s", defaultValue, rid)
		return fmt.Errorf("array_time default value must be []interface{}, got %T", defaultValue)
	}

	// Validate each item in default array
	for i, item := range defaultArray {
		if _, ok := util.IsTime(item); !ok {
			blog.Errorf("array_time default value array item [%d] type %T is not a valid time, rid: %s", i, item, rid)
			return fmt.Errorf("array_time default value array item [%d] is not a valid time", i)
		}
	}

	valData[propertyId] = defaultArray
	return nil
}

// ValidateOption validates the option field
func (a arrayTime) ValidateOption(ctx context.Context, option interface{}, defaultVal interface{}) error {

	rid := util.ExtractRequestIDFromContext(ctx)

	// For time array, option is typically empty or can contain display settings
	// No specific validation needed for option
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
		blog.Errorf("array_time default value must be []interface{}, got %T, rid: %s", defaultVal, rid)
		return fmt.Errorf("array_time default value must be []interface{}, got %T", defaultVal)
	}

	// Validate each item in default array
	for i, item := range defaultArray {
		if _, ok := util.IsTime(item); !ok {
			blog.Errorf("array_time default value array item [%d] type %T is not a valid time, rid: %s", i, item, rid)
			return fmt.Errorf("array_time default value array item [%d] is not a valid time", i)
		}
	}

	return nil
}

var _ register.AttributeTypeI = &arrayTime{}
