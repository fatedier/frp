package validator

import (
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

// BakedInAliasValidators is a default mapping of a single validationstag that
// defines a common or complex set of validation(s) to simplify
// adding validation to structs. i.e. set key "_ageok" and the tags
// are "gt=0,lte=130" or key "_preferredname" and tags "omitempty,gt=0,lte=60"
var bakedInAliasValidators = map[string]string{
	"iscolor": "hexcolor|rgb|rgba|hsl|hsla",
}

// BakedInValidators is the default map of ValidationFunc
// you can add, remove or even replace items to suite your needs,
// or even disregard and use your own map if so desired.
var bakedInValidators = map[string]Func{
	"required":     HasValue,
	"len":          HasLengthOf,
	"min":          HasMinOf,
	"max":          HasMaxOf,
	"eq":           IsEq,
	"ne":           IsNe,
	"lt":           IsLt,
	"lte":          IsLte,
	"gt":           IsGt,
	"gte":          IsGte,
	"eqfield":      IsEqField,
	"eqcsfield":    IsEqCrossStructField,
	"necsfield":    IsNeCrossStructField,
	"gtcsfield":    IsGtCrossStructField,
	"gtecsfield":   IsGteCrossStructField,
	"ltcsfield":    IsLtCrossStructField,
	"ltecsfield":   IsLteCrossStructField,
	"nefield":      IsNeField,
	"gtefield":     IsGteField,
	"gtfield":      IsGtField,
	"ltefield":     IsLteField,
	"ltfield":      IsLtField,
	"alpha":        IsAlpha,
	"alphanum":     IsAlphanum,
	"numeric":      IsNumeric,
	"number":       IsNumber,
	"hexadecimal":  IsHexadecimal,
	"hexcolor":     IsHEXColor,
	"rgb":          IsRGB,
	"rgba":         IsRGBA,
	"hsl":          IsHSL,
	"hsla":         IsHSLA,
	"email":        IsEmail,
	"url":          IsURL,
	"uri":          IsURI,
	"base64":       IsBase64,
	"contains":     Contains,
	"containsany":  ContainsAny,
	"containsrune": ContainsRune,
	"excludes":     Excludes,
	"excludesall":  ExcludesAll,
	"excludesrune": ExcludesRune,
	"isbn":         IsISBN,
	"isbn10":       IsISBN10,
	"isbn13":       IsISBN13,
	"uuid":         IsUUID,
	"uuid3":        IsUUID3,
	"uuid4":        IsUUID4,
	"uuid5":        IsUUID5,
	"ascii":        IsASCII,
	"printascii":   IsPrintableASCII,
	"multibyte":    HasMultiByteCharacter,
	"datauri":      IsDataURI,
	"latitude":     IsLatitude,
	"longitude":    IsLongitude,
	"ssn":          IsSSN,
	"ipv4":         IsIPv4,
	"ipv6":         IsIPv6,
	"ip":           IsIP,
	"cidrv4":       IsCIDRv4,
	"cidrv6":       IsCIDRv6,
	"cidr":         IsCIDR,
	"tcp4_addr":    IsTCP4AddrResolvable,
	"tcp6_addr":    IsTCP6AddrResolvable,
	"tcp_addr":     IsTCPAddrResolvable,
	"udp4_addr":    IsUDP4AddrResolvable,
	"udp6_addr":    IsUDP6AddrResolvable,
	"udp_addr":     IsUDPAddrResolvable,
	"ip4_addr":     IsIP4AddrResolvable,
	"ip6_addr":     IsIP6AddrResolvable,
	"ip_addr":      IsIPAddrResolvable,
	"unix_addr":    IsUnixAddrResolvable,
	"mac":          IsMAC,
}

// IsMAC is the validation function for validating if the field's value is a valid MAC address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsMAC(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	_, err := net.ParseMAC(field.String())
	return err == nil
}

// IsCIDRv4 is the validation function for validating if the field's value is a valid v4 CIDR address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsCIDRv4(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	ip, _, err := net.ParseCIDR(field.String())

	return err == nil && ip.To4() != nil
}

// IsCIDRv6 is the validation function for validating if the field's value is a valid v6 CIDR address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsCIDRv6(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	ip, _, err := net.ParseCIDR(field.String())

	return err == nil && ip.To4() == nil
}

// IsCIDR is the validation function for validating if the field's value is a valid v4 or v6 CIDR address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsCIDR(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	_, _, err := net.ParseCIDR(field.String())

	return err == nil
}

// IsIPv4 is the validation function for validating if a value is a valid v4 IP address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsIPv4(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	ip := net.ParseIP(field.String())

	return ip != nil && ip.To4() != nil
}

// IsIPv6 is the validation function for validating if the field's value is a valid v6 IP address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsIPv6(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	ip := net.ParseIP(field.String())

	return ip != nil && ip.To4() == nil
}

// IsIP is the validation function for validating if the field's value is a valid v4 or v6 IP address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsIP(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	ip := net.ParseIP(field.String())

	return ip != nil
}

// IsSSN is the validation function for validating if the field's value is a valid SSN.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsSSN(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if field.Len() != 11 {
		return false
	}

	return sSNRegex.MatchString(field.String())
}

// IsLongitude is the validation function for validating if the field's value is a valid longitude coordinate.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLongitude(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return longitudeRegex.MatchString(field.String())
}

// IsLatitude is the validation function for validating if the field's value is a valid latitude coordinate.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLatitude(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return latitudeRegex.MatchString(field.String())
}

// IsDataURI is the validation function for validating if the field's value is a valid data URI.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsDataURI(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	uri := strings.SplitN(field.String(), ",", 2)

	if len(uri) != 2 {
		return false
	}

	if !dataURIRegex.MatchString(uri[0]) {
		return false
	}

	fld := reflect.ValueOf(uri[1])

	return IsBase64(v, topStruct, currentStructOrField, fld, fld.Type(), fld.Kind(), param)
}

// HasMultiByteCharacter is the validation function for validating if the field's value has a multi byte character.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func HasMultiByteCharacter(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if field.Len() == 0 {
		return true
	}

	return multibyteRegex.MatchString(field.String())
}

// IsPrintableASCII is the validation function for validating if the field's value is a valid printable ASCII character.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsPrintableASCII(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return printableASCIIRegex.MatchString(field.String())
}

// IsASCII is the validation function for validating if the field's value is a valid ASCII character.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsASCII(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return aSCIIRegex.MatchString(field.String())
}

// IsUUID5 is the validation function for validating if the field's value is a valid v5 UUID.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUUID5(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return uUID5Regex.MatchString(field.String())
}

// IsUUID4 is the validation function for validating if the field's value is a valid v4 UUID.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUUID4(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return uUID4Regex.MatchString(field.String())
}

// IsUUID3 is the validation function for validating if the field's value is a valid v3 UUID.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUUID3(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return uUID3Regex.MatchString(field.String())
}

// IsUUID is the validation function for validating if the field's value is a valid UUID of any version.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUUID(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return uUIDRegex.MatchString(field.String())
}

// IsISBN is the validation function for validating if the field's value is a valid v10 or v13 ISBN.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsISBN(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return IsISBN10(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) || IsISBN13(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param)
}

// IsISBN13 is the validation function for validating if the field's value is a valid v13 ISBN.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsISBN13(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	s := strings.Replace(strings.Replace(field.String(), "-", "", 4), " ", "", 4)

	if !iSBN13Regex.MatchString(s) {
		return false
	}

	var checksum int32
	var i int32

	factor := []int32{1, 3}

	for i = 0; i < 12; i++ {
		checksum += factor[i%2] * int32(s[i]-'0')
	}

	if (int32(s[12]-'0'))-((10-(checksum%10))%10) == 0 {
		return true
	}

	return false
}

// IsISBN10 is the validation function for validating if the field's value is a valid v10 ISBN.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsISBN10(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	s := strings.Replace(strings.Replace(field.String(), "-", "", 3), " ", "", 3)

	if !iSBN10Regex.MatchString(s) {
		return false
	}

	var checksum int32
	var i int32

	for i = 0; i < 9; i++ {
		checksum += (i + 1) * int32(s[i]-'0')
	}

	if s[9] == 'X' {
		checksum += 10 * 10
	} else {
		checksum += 10 * int32(s[9]-'0')
	}

	if checksum%11 == 0 {
		return true
	}

	return false
}

// ExcludesRune is the validation function for validating that the field's value does not contain the rune specified withing the param.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func ExcludesRune(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return !ContainsRune(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param)
}

// ExcludesAll is the validation function for validating that the field's value does not contain any of the characters specified withing the param.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func ExcludesAll(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return !ContainsAny(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param)
}

// Excludes is the validation function for validating that the field's value does not contain the text specified withing the param.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func Excludes(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return !Contains(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param)
}

// ContainsRune is the validation function for validating that the field's value contains the rune specified withing the param.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func ContainsRune(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	r, _ := utf8.DecodeRuneInString(param)

	return strings.ContainsRune(field.String(), r)
}

// ContainsAny is the validation function for validating that the field's value contains any of the characters specified withing the param.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func ContainsAny(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return strings.ContainsAny(field.String(), param)
}

// Contains is the validation function for validating that the field's value contains the text specified withing the param.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func Contains(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return strings.Contains(field.String(), param)
}

// IsNeField is the validation function for validating if the current field's value is not equal to the field specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsNeField(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	currentField, currentKind, ok := v.GetStructFieldOK(currentStructOrField, param)

	if !ok || currentKind != fieldKind {
		return true
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() != currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() != currentField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() != currentField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) != int64(currentField.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return true
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return !fieldTime.Equal(t)
		}

	}

	// default reflect.String:
	return field.String() != currentField.String()
}

// IsNe is the validation function for validating that the field's value does not equal the provided param value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsNe(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return !IsEq(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param)
}

// IsLteCrossStructField is the validation function for validating if the current field's value is less than or equal to the field, within a separate struct, specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLteCrossStructField(v *Validate, topStruct reflect.Value, current reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	topField, topKind, ok := v.GetStructFieldOK(topStruct, param)
	if !ok || topKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() <= topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() <= topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() <= topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) <= int64(topField.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.Before(topTime) || fieldTime.Equal(topTime)
		}
	}

	// default reflect.String:
	return field.String() <= topField.String()
}

// IsLtCrossStructField is the validation function for validating if the current field's value is less than the field, within a separate struct, specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLtCrossStructField(v *Validate, topStruct reflect.Value, current reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	topField, topKind, ok := v.GetStructFieldOK(topStruct, param)
	if !ok || topKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() < topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() < topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() < topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) < int64(topField.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.Before(topTime)
		}
	}

	// default reflect.String:
	return field.String() < topField.String()
}

// IsGteCrossStructField is the validation function for validating if the current field's value is greater than or equal to the field, within a separate struct, specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsGteCrossStructField(v *Validate, topStruct reflect.Value, current reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	topField, topKind, ok := v.GetStructFieldOK(topStruct, param)
	if !ok || topKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() >= topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() >= topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() >= topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) >= int64(topField.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.After(topTime) || fieldTime.Equal(topTime)
		}
	}

	// default reflect.String:
	return field.String() >= topField.String()
}

// IsGtCrossStructField is the validation function for validating if the current field's value is greater than the field, within a separate struct, specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsGtCrossStructField(v *Validate, topStruct reflect.Value, current reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	topField, topKind, ok := v.GetStructFieldOK(topStruct, param)
	if !ok || topKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() > topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() > topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() > topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) > int64(topField.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.After(topTime)
		}
	}

	// default reflect.String:
	return field.String() > topField.String()
}

// IsNeCrossStructField is the validation function for validating that the current field's value is not equal to the field, within a separate struct, specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsNeCrossStructField(v *Validate, topStruct reflect.Value, current reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	topField, currentKind, ok := v.GetStructFieldOK(topStruct, param)
	if !ok || currentKind != fieldKind {
		return true
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return topField.Int() != field.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return topField.Uint() != field.Uint()

	case reflect.Float32, reflect.Float64:
		return topField.Float() != field.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(topField.Len()) != int64(field.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return true
		}

		if fieldType == timeType {

			t := field.Interface().(time.Time)
			fieldTime := topField.Interface().(time.Time)

			return !fieldTime.Equal(t)
		}
	}

	// default reflect.String:
	return topField.String() != field.String()
}

// IsEqCrossStructField is the validation function for validating that the current field's value is equal to the field, within a separate struct, specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsEqCrossStructField(v *Validate, topStruct reflect.Value, current reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	topField, topKind, ok := v.GetStructFieldOK(topStruct, param)
	if !ok || topKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return topField.Int() == field.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return topField.Uint() == field.Uint()

	case reflect.Float32, reflect.Float64:
		return topField.Float() == field.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(topField.Len()) == int64(field.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			t := field.Interface().(time.Time)
			fieldTime := topField.Interface().(time.Time)

			return fieldTime.Equal(t)
		}
	}

	// default reflect.String:
	return topField.String() == field.String()
}

// IsEqField is the validation function for validating if the current field's value is equal to the field specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsEqField(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	currentField, currentKind, ok := v.GetStructFieldOK(currentStructOrField, param)
	if !ok || currentKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() == currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() == currentField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() == currentField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) == int64(currentField.Len())

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.Equal(t)
		}

	}

	// default reflect.String:
	return field.String() == currentField.String()
}

// IsEq is the validation function for validating if the current field's value is equal to the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsEq(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:
		return field.String() == param

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) == p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return field.Int() == p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() == p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() == p
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// IsBase64 is the validation function for validating if the current field's value is a valid base 64.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsBase64(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return base64Regex.MatchString(field.String())
}

// IsURI is the validation function for validating if the current field's value is a valid URI.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsURI(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:

		s := field.String()

		// checks needed as of Go 1.6 because of change https://github.com/golang/go/commit/617c93ce740c3c3cc28cdd1a0d712be183d0b328#diff-6c2d018290e298803c0c9419d8739885L195
		// emulate browser and strip the '#' suffix prior to validation. see issue-#237
		if i := strings.Index(s, "#"); i > -1 {
			s = s[:i]
		}

		if s == blank {
			return false
		}

		_, err := url.ParseRequestURI(s)

		return err == nil
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// IsURL is the validation function for validating if the current field's value is a valid URL.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsURL(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:

		var i int
		s := field.String()

		// checks needed as of Go 1.6 because of change https://github.com/golang/go/commit/617c93ce740c3c3cc28cdd1a0d712be183d0b328#diff-6c2d018290e298803c0c9419d8739885L195
		// emulate browser and strip the '#' suffix prior to validation. see issue-#237
		if i = strings.Index(s, "#"); i > -1 {
			s = s[:i]
		}

		if s == blank {
			return false
		}

		url, err := url.ParseRequestURI(s)

		if err != nil || url.Scheme == blank {
			return false
		}

		return err == nil
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// IsEmail is the validation function for validating if the current field's value is a valid email address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsEmail(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return emailRegex.MatchString(field.String())
}

// IsHSLA is the validation function for validating if the current field's value is a valid HSLA color.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsHSLA(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return hslaRegex.MatchString(field.String())
}

// IsHSL is the validation function for validating if the current field's value is a valid HSL color.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsHSL(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return hslRegex.MatchString(field.String())
}

// IsRGBA is the validation function for validating if the current field's value is a valid RGBA color.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsRGBA(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return rgbaRegex.MatchString(field.String())
}

// IsRGB is the validation function for validating if the current field's value is a valid RGB color.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsRGB(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return rgbRegex.MatchString(field.String())
}

// IsHEXColor is the validation function for validating if the current field's value is a valid HEX color.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsHEXColor(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return hexcolorRegex.MatchString(field.String())
}

// IsHexadecimal is the validation function for validating if the current field's value is a valid hexadecimal.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsHexadecimal(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return hexadecimalRegex.MatchString(field.String())
}

// IsNumber is the validation function for validating if the current field's value is a valid number.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsNumber(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return numberRegex.MatchString(field.String())
}

// IsNumeric is the validation function for validating if the current field's value is a valid numeric value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsNumeric(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return numericRegex.MatchString(field.String())
}

// IsAlphanum is the validation function for validating if the current field's value is a valid alphanumeric value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsAlphanum(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return alphaNumericRegex.MatchString(field.String())
}

// IsAlpha is the validation function for validating if the current field's value is a valid alpha value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsAlpha(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return alphaRegex.MatchString(field.String())
}

// HasValue is the validation function for validating if the current field's value is not the default static value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func HasValue(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return !field.IsNil()
	default:
		return field.IsValid() && field.Interface() != reflect.Zero(fieldType).Interface()
	}
}

// IsGteField is the validation function for validating if the current field's value is greater than or equal to the field specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsGteField(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	currentField, currentKind, ok := v.GetStructFieldOK(currentStructOrField, param)
	if !ok || currentKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() >= currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() >= currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() >= currentField.Float()

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.After(t) || fieldTime.Equal(t)
		}
	}

	// default reflect.String
	return len(field.String()) >= len(currentField.String())
}

// IsGtField is the validation function for validating if the current field's value is greater than the field specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsGtField(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	currentField, currentKind, ok := v.GetStructFieldOK(currentStructOrField, param)
	if !ok || currentKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() > currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() > currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() > currentField.Float()

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.After(t)
		}
	}

	// default reflect.String
	return len(field.String()) > len(currentField.String())
}

// IsGte is the validation function for validating if the current field's value is greater than or equal to the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsGte(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) >= p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) >= p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return field.Int() >= p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() >= p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() >= p

	case reflect.Struct:

		if fieldType == timeType || fieldType == timePtrType {

			now := time.Now().UTC()
			t := field.Interface().(time.Time)

			return t.After(now) || t.Equal(now)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// IsGt is the validation function for validating if the current field's value is greater than the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsGt(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) > p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) > p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return field.Int() > p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() > p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() > p
	case reflect.Struct:

		if field.Type() == timeType || field.Type() == timePtrType {

			return field.Interface().(time.Time).After(time.Now().UTC())
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// HasLengthOf is the validation function for validating if the current field's value is equal to the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func HasLengthOf(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) == p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) == p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return field.Int() == p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() == p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() == p
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// HasMinOf is the validation function for validating if the current field's value is greater than or equal to the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func HasMinOf(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	return IsGte(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param)
}

// IsLteField is the validation function for validating if the current field's value is less than or equal to the field specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLteField(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	currentField, currentKind, ok := v.GetStructFieldOK(currentStructOrField, param)
	if !ok || currentKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() <= currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() <= currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() <= currentField.Float()

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.Before(t) || fieldTime.Equal(t)
		}
	}

	// default reflect.String
	return len(field.String()) <= len(currentField.String())
}

// IsLtField is the validation function for validating if the current field's value is less than the field specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLtField(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	currentField, currentKind, ok := v.GetStructFieldOK(currentStructOrField, param)
	if !ok || currentKind != fieldKind {
		return false
	}

	switch fieldKind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() < currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() < currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() < currentField.Float()

	case reflect.Struct:

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.Before(t)
		}
	}

	// default reflect.String
	return len(field.String()) < len(currentField.String())
}

// IsLte is the validation function for validating if the current field's value is less than or equal to the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLte(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) <= p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) <= p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return field.Int() <= p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() <= p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() <= p

	case reflect.Struct:

		if fieldType == timeType || fieldType == timePtrType {

			now := time.Now().UTC()
			t := field.Interface().(time.Time)

			return t.Before(now) || t.Equal(now)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// IsLt is the validation function for validating if the current field's value is less than the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsLt(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	switch fieldKind {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) < p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) < p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asInt(param)

		return field.Int() < p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() < p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() < p

	case reflect.Struct:

		if field.Type() == timeType || field.Type() == timePtrType {

			return field.Interface().(time.Time).Before(time.Now().UTC())
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// HasMaxOf is the validation function for validating if the current field's value is less than or equal to the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func HasMaxOf(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	return IsLte(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param)
}

// IsTCP4AddrResolvable is the validation function for validating if the field's value is a resolvable tcp4 address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsTCP4AddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !isIP4Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp4", field.String())
	return err == nil
}

// IsTCP6AddrResolvable is the validation function for validating if the field's value is a resolvable tcp6 address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsTCP6AddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !isIP6Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp6", field.String())
	return err == nil
}

// IsTCPAddrResolvable is the validation function for validating if the field's value is a resolvable tcp address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsTCPAddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !isIP4Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) &&
		!isIP6Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp", field.String())
	return err == nil
}

// IsUDP4AddrResolvable is the validation function for validating if the field's value is a resolvable udp4 address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUDP4AddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !isIP4Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp4", field.String())
	return err == nil
}

// IsUDP6AddrResolvable is the validation function for validating if the field's value is a resolvable udp6 address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUDP6AddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !isIP6Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp6", field.String())
	return err == nil
}

// IsUDPAddrResolvable is the validation function for validating if the field's value is a resolvable udp address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUDPAddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !isIP4Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) &&
		!isIP6Addr(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp", field.String())
	return err == nil
}

// IsIP4AddrResolvable is the validation function for validating if the field's value is a resolvable ip4 address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsIP4AddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !IsIPv4(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveIPAddr("ip4", field.String())
	return err == nil
}

// IsIP6AddrResolvable is the validation function for validating if the field's value is a resolvable ip6 address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsIP6AddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !IsIPv6(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveIPAddr("ip6", field.String())
	return err == nil
}

// IsIPAddrResolvable is the validation function for validating if the field's value is a resolvable ip address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsIPAddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

	if !IsIP(v, topStruct, currentStructOrField, field, fieldType, fieldKind, param) {
		return false
	}

	_, err := net.ResolveIPAddr("ip", field.String())
	return err == nil
}

// IsUnixAddrResolvable is the validation function for validating if the field's value is a resolvable unix address.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func IsUnixAddrResolvable(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	_, err := net.ResolveUnixAddr("unix", field.String())
	return err == nil
}

func isIP4Addr(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	val := field.String()

	if idx := strings.LastIndex(val, ":"); idx != -1 {
		val = val[0:idx]
	}

	if !IsIPv4(v, topStruct, currentStructOrField, reflect.ValueOf(val), fieldType, fieldKind, param) {
		return false
	}

	return true
}

func isIP6Addr(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {
	val := field.String()

	if idx := strings.LastIndex(val, ":"); idx != -1 {
		if idx != 0 && val[idx-1:idx] == "]" {
			val = val[1 : idx-1]
		}
	}

	if !IsIPv6(v, topStruct, currentStructOrField, reflect.ValueOf(val), fieldType, fieldKind, param) {
		return false
	}

	return true
}
