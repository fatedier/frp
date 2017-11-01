// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package orm

import (
	"fmt"
	"strconv"
	"time"
)

// Define the Type enum
const (
	TypeBooleanField = 1 << iota
	TypeCharField
	TypeTextField
	TypeTimeField
	TypeDateField
	TypeDateTimeField
	TypeBitField
	TypeSmallIntegerField
	TypeIntegerField
	TypeBigIntegerField
	TypePositiveBitField
	TypePositiveSmallIntegerField
	TypePositiveIntegerField
	TypePositiveBigIntegerField
	TypeFloatField
	TypeDecimalField
	TypeJSONField
	TypeJsonbField
	RelForeignKey
	RelOneToOne
	RelManyToMany
	RelReverseOne
	RelReverseMany
)

// Define some logic enum
const (
	IsIntegerField         = ^-TypePositiveBigIntegerField >> 5 << 6
	IsPositiveIntegerField = ^-TypePositiveBigIntegerField >> 9 << 10
	IsRelField             = ^-RelReverseMany >> 17 << 18
	IsFieldType            = ^-RelReverseMany<<1 + 1
)

// BooleanField A true/false field.
type BooleanField bool

// Value return the BooleanField
func (e BooleanField) Value() bool {
	return bool(e)
}

// Set will set the BooleanField
func (e *BooleanField) Set(d bool) {
	*e = BooleanField(d)
}

// String format the Bool to string
func (e *BooleanField) String() string {
	return strconv.FormatBool(e.Value())
}

// FieldType return BooleanField the type
func (e *BooleanField) FieldType() int {
	return TypeBooleanField
}

// SetRaw set the interface to bool
func (e *BooleanField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case bool:
		e.Set(d)
	case string:
		v, err := StrTo(d).Bool()
		if err != nil {
			e.Set(v)
		}
		return err
	default:
		return fmt.Errorf("<BooleanField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return the current value
func (e *BooleanField) RawValue() interface{} {
	return e.Value()
}

// verify the BooleanField implement the Fielder interface
var _ Fielder = new(BooleanField)

// CharField A string field
// required values tag: size
// The size is enforced at the database level and in models’s validation.
// eg: `orm:"size(120)"`
type CharField string

// Value return the CharField's Value
func (e CharField) Value() string {
	return string(e)
}

// Set CharField value
func (e *CharField) Set(d string) {
	*e = CharField(d)
}

// String return the CharField
func (e *CharField) String() string {
	return e.Value()
}

// FieldType return the enum type
func (e *CharField) FieldType() int {
	return TypeCharField
}

// SetRaw set the interface to string
func (e *CharField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case string:
		e.Set(d)
	default:
		return fmt.Errorf("<CharField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return the CharField value
func (e *CharField) RawValue() interface{} {
	return e.Value()
}

// verify CharField implement Fielder
var _ Fielder = new(CharField)

// TimeField A time, represented in go by a time.Time instance.
// only time values like 10:00:00
// Has a few extra, optional attr tag:
//
// auto_now:
// Automatically set the field to now every time the object is saved. Useful for “last-modified” timestamps.
// Note that the current date is always used; it’s not just a default value that you can override.
//
// auto_now_add:
// Automatically set the field to now when the object is first created. Useful for creation of timestamps.
// Note that the current date is always used; it’s not just a default value that you can override.
//
// eg: `orm:"auto_now"` or `orm:"auto_now_add"`
type TimeField time.Time

// Value return the time.Time
func (e TimeField) Value() time.Time {
	return time.Time(e)
}

// Set set the TimeField's value
func (e *TimeField) Set(d time.Time) {
	*e = TimeField(d)
}

// String convert time to string
func (e *TimeField) String() string {
	return e.Value().String()
}

// FieldType return enum type Date
func (e *TimeField) FieldType() int {
	return TypeDateField
}

// SetRaw convert the interface to time.Time. Allow string and time.Time
func (e *TimeField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case time.Time:
		e.Set(d)
	case string:
		v, err := timeParse(d, formatTime)
		if err != nil {
			e.Set(v)
		}
		return err
	default:
		return fmt.Errorf("<TimeField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return time value
func (e *TimeField) RawValue() interface{} {
	return e.Value()
}

var _ Fielder = new(TimeField)

// DateField A date, represented in go by a time.Time instance.
// only date values like 2006-01-02
// Has a few extra, optional attr tag:
//
// auto_now:
// Automatically set the field to now every time the object is saved. Useful for “last-modified” timestamps.
// Note that the current date is always used; it’s not just a default value that you can override.
//
// auto_now_add:
// Automatically set the field to now when the object is first created. Useful for creation of timestamps.
// Note that the current date is always used; it’s not just a default value that you can override.
//
// eg: `orm:"auto_now"` or `orm:"auto_now_add"`
type DateField time.Time

// Value return the time.Time
func (e DateField) Value() time.Time {
	return time.Time(e)
}

// Set set the DateField's value
func (e *DateField) Set(d time.Time) {
	*e = DateField(d)
}

// String convert datatime to string
func (e *DateField) String() string {
	return e.Value().String()
}

// FieldType return enum type Date
func (e *DateField) FieldType() int {
	return TypeDateField
}

// SetRaw convert the interface to time.Time. Allow string and time.Time
func (e *DateField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case time.Time:
		e.Set(d)
	case string:
		v, err := timeParse(d, formatDate)
		if err != nil {
			e.Set(v)
		}
		return err
	default:
		return fmt.Errorf("<DateField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return Date value
func (e *DateField) RawValue() interface{} {
	return e.Value()
}

// verify DateField implement fielder interface
var _ Fielder = new(DateField)

// DateTimeField A date, represented in go by a time.Time instance.
// datetime values like 2006-01-02 15:04:05
// Takes the same extra arguments as DateField.
type DateTimeField time.Time

// Value return the datatime value
func (e DateTimeField) Value() time.Time {
	return time.Time(e)
}

// Set set the time.Time to datatime
func (e *DateTimeField) Set(d time.Time) {
	*e = DateTimeField(d)
}

// String return the time's String
func (e *DateTimeField) String() string {
	return e.Value().String()
}

// FieldType return the enum TypeDateTimeField
func (e *DateTimeField) FieldType() int {
	return TypeDateTimeField
}

// SetRaw convert the string or time.Time to DateTimeField
func (e *DateTimeField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case time.Time:
		e.Set(d)
	case string:
		v, err := timeParse(d, formatDateTime)
		if err != nil {
			e.Set(v)
		}
		return err
	default:
		return fmt.Errorf("<DateTimeField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return the datatime value
func (e *DateTimeField) RawValue() interface{} {
	return e.Value()
}

// verify datatime implement fielder
var _ Fielder = new(DateTimeField)

// FloatField A floating-point number represented in go by a float32 value.
type FloatField float64

// Value return the FloatField value
func (e FloatField) Value() float64 {
	return float64(e)
}

// Set the Float64
func (e *FloatField) Set(d float64) {
	*e = FloatField(d)
}

// String return the string
func (e *FloatField) String() string {
	return ToStr(e.Value(), -1, 32)
}

// FieldType return the enum type
func (e *FloatField) FieldType() int {
	return TypeFloatField
}

// SetRaw converter interface Float64 float32 or string to FloatField
func (e *FloatField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case float32:
		e.Set(float64(d))
	case float64:
		e.Set(d)
	case string:
		v, err := StrTo(d).Float64()
		if err != nil {
			e.Set(v)
		}
	default:
		return fmt.Errorf("<FloatField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return the FloatField value
func (e *FloatField) RawValue() interface{} {
	return e.Value()
}

// verify FloatField implement Fielder
var _ Fielder = new(FloatField)

// SmallIntegerField -32768 to 32767
type SmallIntegerField int16

// Value return int16 value
func (e SmallIntegerField) Value() int16 {
	return int16(e)
}

// Set the SmallIntegerField value
func (e *SmallIntegerField) Set(d int16) {
	*e = SmallIntegerField(d)
}

// String convert smallint to string
func (e *SmallIntegerField) String() string {
	return ToStr(e.Value())
}

// FieldType return enum type SmallIntegerField
func (e *SmallIntegerField) FieldType() int {
	return TypeSmallIntegerField
}

// SetRaw convert interface int16/string to int16
func (e *SmallIntegerField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case int16:
		e.Set(d)
	case string:
		v, err := StrTo(d).Int16()
		if err != nil {
			e.Set(v)
		}
	default:
		return fmt.Errorf("<SmallIntegerField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return smallint value
func (e *SmallIntegerField) RawValue() interface{} {
	return e.Value()
}

// verify SmallIntegerField implement Fielder
var _ Fielder = new(SmallIntegerField)

// IntegerField -2147483648 to 2147483647
type IntegerField int32

// Value return the int32
func (e IntegerField) Value() int32 {
	return int32(e)
}

// Set IntegerField value
func (e *IntegerField) Set(d int32) {
	*e = IntegerField(d)
}

// String convert Int32 to string
func (e *IntegerField) String() string {
	return ToStr(e.Value())
}

// FieldType return the enum type
func (e *IntegerField) FieldType() int {
	return TypeIntegerField
}

// SetRaw convert interface int32/string to int32
func (e *IntegerField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case int32:
		e.Set(d)
	case string:
		v, err := StrTo(d).Int32()
		if err != nil {
			e.Set(v)
		}
	default:
		return fmt.Errorf("<IntegerField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return IntegerField value
func (e *IntegerField) RawValue() interface{} {
	return e.Value()
}

// verify IntegerField implement Fielder
var _ Fielder = new(IntegerField)

// BigIntegerField -9223372036854775808 to 9223372036854775807.
type BigIntegerField int64

// Value return int64
func (e BigIntegerField) Value() int64 {
	return int64(e)
}

// Set the BigIntegerField value
func (e *BigIntegerField) Set(d int64) {
	*e = BigIntegerField(d)
}

// String convert BigIntegerField to string
func (e *BigIntegerField) String() string {
	return ToStr(e.Value())
}

// FieldType return enum type
func (e *BigIntegerField) FieldType() int {
	return TypeBigIntegerField
}

// SetRaw convert interface int64/string to int64
func (e *BigIntegerField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case int64:
		e.Set(d)
	case string:
		v, err := StrTo(d).Int64()
		if err != nil {
			e.Set(v)
		}
	default:
		return fmt.Errorf("<BigIntegerField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return BigIntegerField value
func (e *BigIntegerField) RawValue() interface{} {
	return e.Value()
}

// verify BigIntegerField implement Fielder
var _ Fielder = new(BigIntegerField)

// PositiveSmallIntegerField 0 to 65535
type PositiveSmallIntegerField uint16

// Value return uint16
func (e PositiveSmallIntegerField) Value() uint16 {
	return uint16(e)
}

// Set PositiveSmallIntegerField value
func (e *PositiveSmallIntegerField) Set(d uint16) {
	*e = PositiveSmallIntegerField(d)
}

// String convert uint16 to string
func (e *PositiveSmallIntegerField) String() string {
	return ToStr(e.Value())
}

// FieldType return enum type
func (e *PositiveSmallIntegerField) FieldType() int {
	return TypePositiveSmallIntegerField
}

// SetRaw convert Interface uint16/string to uint16
func (e *PositiveSmallIntegerField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case uint16:
		e.Set(d)
	case string:
		v, err := StrTo(d).Uint16()
		if err != nil {
			e.Set(v)
		}
	default:
		return fmt.Errorf("<PositiveSmallIntegerField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue returns PositiveSmallIntegerField value
func (e *PositiveSmallIntegerField) RawValue() interface{} {
	return e.Value()
}

// verify PositiveSmallIntegerField implement Fielder
var _ Fielder = new(PositiveSmallIntegerField)

// PositiveIntegerField 0 to 4294967295
type PositiveIntegerField uint32

// Value return PositiveIntegerField value. Uint32
func (e PositiveIntegerField) Value() uint32 {
	return uint32(e)
}

// Set the PositiveIntegerField value
func (e *PositiveIntegerField) Set(d uint32) {
	*e = PositiveIntegerField(d)
}

// String convert PositiveIntegerField to string
func (e *PositiveIntegerField) String() string {
	return ToStr(e.Value())
}

// FieldType return enum type
func (e *PositiveIntegerField) FieldType() int {
	return TypePositiveIntegerField
}

// SetRaw convert interface uint32/string to Uint32
func (e *PositiveIntegerField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case uint32:
		e.Set(d)
	case string:
		v, err := StrTo(d).Uint32()
		if err != nil {
			e.Set(v)
		}
	default:
		return fmt.Errorf("<PositiveIntegerField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return the PositiveIntegerField Value
func (e *PositiveIntegerField) RawValue() interface{} {
	return e.Value()
}

// verify PositiveIntegerField implement Fielder
var _ Fielder = new(PositiveIntegerField)

// PositiveBigIntegerField 0 to 18446744073709551615
type PositiveBigIntegerField uint64

// Value return uint64
func (e PositiveBigIntegerField) Value() uint64 {
	return uint64(e)
}

// Set PositiveBigIntegerField value
func (e *PositiveBigIntegerField) Set(d uint64) {
	*e = PositiveBigIntegerField(d)
}

// String convert PositiveBigIntegerField to string
func (e *PositiveBigIntegerField) String() string {
	return ToStr(e.Value())
}

// FieldType return enum type
func (e *PositiveBigIntegerField) FieldType() int {
	return TypePositiveIntegerField
}

// SetRaw convert interface uint64/string to Uint64
func (e *PositiveBigIntegerField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case uint64:
		e.Set(d)
	case string:
		v, err := StrTo(d).Uint64()
		if err != nil {
			e.Set(v)
		}
	default:
		return fmt.Errorf("<PositiveBigIntegerField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return PositiveBigIntegerField value
func (e *PositiveBigIntegerField) RawValue() interface{} {
	return e.Value()
}

// verify PositiveBigIntegerField implement Fielder
var _ Fielder = new(PositiveBigIntegerField)

// TextField A large text field.
type TextField string

// Value return TextField value
func (e TextField) Value() string {
	return string(e)
}

// Set the TextField value
func (e *TextField) Set(d string) {
	*e = TextField(d)
}

// String convert TextField to string
func (e *TextField) String() string {
	return e.Value()
}

// FieldType return enum type
func (e *TextField) FieldType() int {
	return TypeTextField
}

// SetRaw convert interface string to string
func (e *TextField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case string:
		e.Set(d)
	default:
		return fmt.Errorf("<TextField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return TextField value
func (e *TextField) RawValue() interface{} {
	return e.Value()
}

// verify TextField implement Fielder
var _ Fielder = new(TextField)

// JSONField postgres json field.
type JSONField string

// Value return JSONField value
func (j JSONField) Value() string {
	return string(j)
}

// Set the JSONField value
func (j *JSONField) Set(d string) {
	*j = JSONField(d)
}

// String convert JSONField to string
func (j *JSONField) String() string {
	return j.Value()
}

// FieldType return enum type
func (j *JSONField) FieldType() int {
	return TypeJSONField
}

// SetRaw convert interface string to string
func (j *JSONField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case string:
		j.Set(d)
	default:
		return fmt.Errorf("<JSONField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return JSONField value
func (j *JSONField) RawValue() interface{} {
	return j.Value()
}

// verify JSONField implement Fielder
var _ Fielder = new(JSONField)

// JsonbField postgres json field.
type JsonbField string

// Value return JsonbField value
func (j JsonbField) Value() string {
	return string(j)
}

// Set the JsonbField value
func (j *JsonbField) Set(d string) {
	*j = JsonbField(d)
}

// String convert JsonbField to string
func (j *JsonbField) String() string {
	return j.Value()
}

// FieldType return enum type
func (j *JsonbField) FieldType() int {
	return TypeJsonbField
}

// SetRaw convert interface string to string
func (j *JsonbField) SetRaw(value interface{}) error {
	switch d := value.(type) {
	case string:
		j.Set(d)
	default:
		return fmt.Errorf("<JsonbField.SetRaw> unknown value `%s`", value)
	}
	return nil
}

// RawValue return JsonbField value
func (j *JsonbField) RawValue() interface{} {
	return j.Value()
}

// verify JsonbField implement Fielder
var _ Fielder = new(JsonbField)
