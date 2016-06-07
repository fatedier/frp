/*
Package validator implements value validations for structs and individual fields
based on tags.

It can also handle Cross-Field and Cross-Struct validation for nested structs
and has the ability to dive into arrays and maps of any type.

Why not a better error message?
Because this library intends for you to handle your own error messages.

Why should I handle my own errors?
Many reasons. We built an internationalized application and needed to know the
field, and what validation failed so we could provide a localized error.

	if fieldErr.Field == "Name" {
		switch fieldErr.ErrorTag
		case "required":
			return "Translated string based on field + error"
		default:
		return "Translated string based on field"
	}


Validation Functions Return Type error

Doing things this way is actually the way the standard library does, see the
file.Open method here:

	https://golang.org/pkg/os/#Open.

The authors return type "error" to avoid the issue discussed in the following,
where err is always != nil:

	http://stackoverflow.com/a/29138676/3158232
	https://github.com/go-playground/validator/issues/134

Validator only returns nil or ValidationErrors as type error; so, in your code
all you need to do is check if the error returned is not nil, and if it's not
type cast it to type ValidationErrors like so err.(validator.ValidationErrors).

Custom Functions

Custom functions can be added. Example:

	// Structure
	func customFunc(v *Validate, topStruct reflect.Value, currentStructOrField reflect.Value, field reflect.Value, fieldType reflect.Type, fieldKind reflect.Kind, param string) bool {

		if whatever {
			return false
		}

		return true
	}

	validate.RegisterValidation("custom tag name", customFunc)
	// NOTES: using the same tag name as an existing function
	//        will overwrite the existing one

Cross-Field Validation

Cross-Field Validation can be done via the following tags:
	- eqfield
	- nefield
	- gtfield
	- gtefield
	- ltfield
	- ltefield
	- eqcsfield
	- necsfield
	- gtcsfield
	- ftecsfield
	- ltcsfield
	- ltecsfield

If, however, some custom cross-field validation is required, it can be done
using a custom validation.

Why not just have cross-fields validation tags (i.e. only eqcsfield and not
eqfield)?

The reason is efficiency. If you want to check a field within the same struct
"eqfield" only has to find the field on the same struct (1 level). But, if we
used "eqcsfield" it could be multiple levels down. Example:

	type Inner struct {
		StartDate time.Time
	}

	type Outer struct {
		InnerStructField *Inner
		CreatedAt time.Time      `validate:"ltecsfield=InnerStructField.StartDate"`
	}

	now := time.Now()

	inner := &Inner{
		StartDate: now,
	}

	outer := &Outer{
		InnerStructField: inner,
		CreatedAt: now,
	}

	errs := validate.Struct(outer)

	// NOTE: when calling validate.Struct(val) topStruct will be the top level struct passed
	//       into the function
	//       when calling validate.FieldWithValue(val, field, tag) val will be
	//       whatever you pass, struct, field...
	//       when calling validate.Field(field, tag) val will be nil

Multiple Validators

Multiple validators on a field will process in the order defined. Example:

	type Test struct {
		Field `validate:"max=10,min=1"`
	}

	// max will be checked then min

Bad Validator definitions are not handled by the library. Example:

	type Test struct {
		Field `validate:"min=10,max=0"`
	}

	// this definition of min max will never succeed

Using Validator Tags

Baked In Cross-Field validation only compares fields on the same struct.
If Cross-Field + Cross-Struct validation is needed you should implement your
own custom validator.

Comma (",") is the default separator of validation tags. If you wish to
have a comma included within the parameter (i.e. excludesall=,) you will need to
use the UTF-8 hex representation 0x2C, which is replaced in the code as a comma,
so the above will become excludesall=0x2C.

	type Test struct {
		Field `validate:"excludesall=,"`    // BAD! Do not include a comma.
		Field `validate:"excludesall=0x2C"` // GOOD! Use the UTF-8 hex representation.
	}

Pipe ("|") is the default separator of validation tags. If you wish to
have a pipe included within the parameter i.e. excludesall=| you will need to
use the UTF-8 hex representation 0x7C, which is replaced in the code as a pipe,
so the above will become excludesall=0x7C

	type Test struct {
		Field `validate:"excludesall=|"`    // BAD! Do not include a a pipe!
		Field `validate:"excludesall=0x7C"` // GOOD! Use the UTF-8 hex representation.
	}


Baked In Validators and Tags

Here is a list of the current built in validators:


Skip Field

Tells the validation to skip this struct field; this is particularily
handy in ignoring embedded structs from being validated. (Usage: -)
	Usage: -


Or Operator

This is the 'or' operator allowing multiple validators to be used and
accepted. (Usage: rbg|rgba) <-- this would allow either rgb or rgba
colors to be accepted. This can also be combined with 'and' for example
( Usage: omitempty,rgb|rgba)

	Usage: |

StructOnly

When a field that is a nested struct is encountered, and contains this flag
any validation on the nested struct will be run, but none of the nested
struct fields will be validated. This is usefull if inside of you program
you know the struct will be valid, but need to verify it has been assigned.
NOTE: only "required" and "omitempty" can be used on a struct itself.

	Usage: structonly

NoStructLevel

Same as structonly tag except that any struct level validations will not run.

	Usage: nostructlevel

Exists

Is a special tag without a validation function attached. It is used when a field
is a Pointer, Interface or Invalid and you wish to validate that it exists.
Example: want to ensure a bool exists if you define the bool as a pointer and
use exists it will ensure there is a value; couldn't use required as it would
fail when the bool was false. exists will fail is the value is a Pointer, Interface
or Invalid and is nil.

	Usage: exists

Omit Empty

Allows conditional validation, for example if a field is not set with
a value (Determined by the "required" validator) then other validation
such as min or max won't run, but if a value is set validation will run.

	Usage: omitempty

Dive

This tells the validator to dive into a slice, array or map and validate that
level of the slice, array or map with the validation tags that follow.
Multidimensional nesting is also supported, each level you wish to dive will
require another dive tag.

	Usage: dive

Example #1

	[][]string with validation tag "gt=0,dive,len=1,dive,required"
	// gt=0 will be applied to []
	// len=1 will be applied to []string
	// required will be applied to string

Example #2

	[][]string with validation tag "gt=0,dive,dive,required"
	// gt=0 will be applied to []
	// []string will be spared validation
	// required will be applied to string

Required

This validates that the value is not the data types default zero value.
For numbers ensures value is not zero. For strings ensures value is
not "". For slices, maps, pointers, interfaces, channels and functions
ensures the value is not nil.

	Usage: required

Length

For numbers, max will ensure that the value is
equal to the parameter given. For strings, it checks that
the string length is exactly that number of characters. For slices,
arrays, and maps, validates the number of items.

	Usage: len=10

Maximum

For numbers, max will ensure that the value is
less than or equal to the parameter given. For strings, it checks
that the string length is at most that number of characters. For
slices, arrays, and maps, validates the number of items.

	Usage: max=10

Mininum

For numbers, min will ensure that the value is
greater or equal to the parameter given. For strings, it checks that
the string length is at least that number of characters. For slices,
arrays, and maps, validates the number of items.

	Usage: min=10

Equals

For strings & numbers, eq will ensure that the value is
equal to the parameter given. For slices, arrays, and maps,
validates the number of items.

	Usage: eq=10

Not Equal

For strings & numbers, eq will ensure that the value is not
equal to the parameter given. For slices, arrays, and maps,
validates the number of items.

	Usage: eq=10

Greater Than

For numbers, this will ensure that the value is greater than the
parameter given. For strings, it checks that the string length
is greater than that number of characters. For slices, arrays
and maps it validates the number of items.

Example #1

	Usage: gt=10

Example #2 (time.Time)

For time.Time ensures the time value is greater than time.Now.UTC().

	Usage: gt

Greater Than or Equal

Same as 'min' above. Kept both to make terminology with 'len' easier.


Example #1

	Usage: gte=10

Example #2 (time.Time)

For time.Time ensures the time value is greater than or equal to time.Now.UTC().

	Usage: gte

Less Than

For numbers, this will ensure that the value is less than the parameter given.
For strings, it checks that the string length is less than that number of
characters. For slices, arrays, and maps it validates the number of items.

Example #1

	Usage: lt=10

Example #2 (time.Time)
For time.Time ensures the time value is less than time.Now.UTC().

	Usage: lt

Less Than or Equal

Same as 'max' above. Kept both to make terminology with 'len' easier.

Example #1

	Usage: lte=10

Example #2 (time.Time)

For time.Time ensures the time value is less than or equal to time.Now.UTC().

	Usage: lte

Field Equals Another Field

This will validate the field value against another fields value either within
a struct or passed in field.

Example #1:

	// Validation on Password field using:
	Usage: eqfield=ConfirmPassword

Example #2:

	// Validating by field:
	validate.FieldWithValue(password, confirmpassword, "eqfield")

Field Equals Another Field (relative)

This does the same as eqfield except that it validates the field provided relative
to the top level struct.

	Usage: eqcsfield=InnerStructField.Field)

Field Does Not Equal Another Field

This will validate the field value against another fields value either within
a struct or passed in field.

Examples:

	// Confirm two colors are not the same:
	//
	// Validation on Color field:
	Usage: nefield=Color2

	// Validating by field:
	validate.FieldWithValue(color1, color2, "nefield")

Field Does Not Equal Another Field (relative)

This does the same as nefield except that it validates the field provided
relative to the top level struct.

	Usage: necsfield=InnerStructField.Field

Field Greater Than Another Field

Only valid for Numbers and time.Time types, this will validate the field value
against another fields value either within a struct or passed in field.
usage examples are for validation of a Start and End date:

Example #1:

	// Validation on End field using:
	validate.Struct Usage(gtfield=Start)

Example #2:

	// Validating by field:
	validate.FieldWithValue(start, end, "gtfield")


Field Greater Than Another Relative Field

This does the same as gtfield except that it validates the field provided
relative to the top level struct.

	Usage: gtcsfield=InnerStructField.Field

Field Greater Than or Equal To Another Field

Only valid for Numbers and time.Time types, this will validate the field value
against another fields value either within a struct or passed in field.
usage examples are for validation of a Start and End date:

Example #1:

	// Validation on End field using:
	validate.Struct Usage(gtefield=Start)

Example #2:

	// Validating by field:
	validate.FieldWithValue(start, end, "gtefield")

Field Greater Than or Equal To Another Relative Field

This does the same as gtefield except that it validates the field provided relative
to the top level struct.

	Usage: gtecsfield=InnerStructField.Field

Less Than Another Field

Only valid for Numbers and time.Time types, this will validate the field value
against another fields value either within a struct or passed in field.
usage examples are for validation of a Start and End date:

Example #1:

	// Validation on End field using:
	validate.Struct Usage(ltfield=Start)

Example #2:

	// Validating by field:
	validate.FieldWithValue(start, end, "ltfield")

Less Than Another Relative Field

This does the same as ltfield except that it validates the field provided relative
to the top level struct.

	Usage: ltcsfield=InnerStructField.Field

Less Than or Equal To Another Field

Only valid for Numbers and time.Time types, this will validate the field value
against another fields value either within a struct or passed in field.
usage examples are for validation of a Start and End date:

Example #1:

	// Validation on End field using:
	validate.Struct Usage(ltefield=Start)

Example #2:

	// Validating by field:
	validate.FieldWithValue(start, end, "ltefield")

Less Than or Equal To Another Relative Field

This does the same as ltefield except that it validates the field provided relative
to the top level struct.

	Usage: ltecsfield=InnerStructField.Field

Alpha Only

This validates that a string value contains alpha characters only

	Usage: alpha

Alphanumeric

This validates that a string value contains alphanumeric characters only

	Usage: alphanum

Numeric

This validates that a string value contains a basic numeric value.
basic excludes exponents etc...

	Usage: numeric

Hexadecimal String

This validates that a string value contains a valid hexadecimal.

	Usage: hexadecimal

Hexcolor String

This validates that a string value contains a valid hex color including
hashtag (#)

		Usage: hexcolor

RGB String

This validates that a string value contains a valid rgb color

	Usage: rgb

RGBA String

This validates that a string value contains a valid rgba color

	Usage: rgba

HSL String

This validates that a string value contains a valid hsl color

	Usage: hsl

HSLA String

This validates that a string value contains a valid hsla color

	Usage: hsla

E-mail String

This validates that a string value contains a valid email
This may not conform to all possibilities of any rfc standard, but neither
does any email provider accept all posibilities.

	Usage: email

URL String

This validates that a string value contains a valid url
This will accept any url the golang request uri accepts but must contain
a schema for example http:// or rtmp://

	Usage: url

URI String

This validates that a string value contains a valid uri
This will accept any uri the golang request uri accepts

	Usage: uri

Base64 String

This validates that a string value contains a valid base64 value.
Although an empty string is valid base64 this will report an empty string
as an error, if you wish to accept an empty string as valid you can use
this with the omitempty tag.

	Usage: base64

Contains

This validates that a string value contains the substring value.

	Usage: contains=@

Contains Any

This validates that a string value contains any Unicode code points
in the substring value.

	Usage: containsany=!@#?

Contains Rune

This validates that a string value contains the supplied rune value.

	Usage: containsrune=@

Excludes

This validates that a string value does not contain the substring value.

	Usage: excludes=@

Excludes All

This validates that a string value does not contain any Unicode code
points in the substring value.

	Usage: excludesall=!@#?

Excludes Rune

This validates that a string value does not contain the supplied rune value.

	Usage: excludesrune=@

International Standard Book Number

This validates that a string value contains a valid isbn10 or isbn13 value.

	Usage: isbn

International Standard Book Number 10

This validates that a string value contains a valid isbn10 value.

	Usage: isbn10

International Standard Book Number 13

This validates that a string value contains a valid isbn13 value.

	Usage: isbn13


Universally Unique Identifier UUID

This validates that a string value contains a valid UUID.

	Usage: uuid

Universally Unique Identifier UUID v3

This validates that a string value contains a valid version 3 UUID.

	Usage: uuid3

Universally Unique Identifier UUID v4

This validates that a string value contains a valid version 4 UUID.

	Usage: uuid4

Universally Unique Identifier UUID v5

This validates that a string value contains a valid version 5 UUID.

	Usage: uuid5

ASCII

This validates that a string value contains only ASCII characters.
NOTE: if the string is blank, this validates as true.

	Usage: ascii

Printable ASCII

This validates that a string value contains only printable ASCII characters.
NOTE: if the string is blank, this validates as true.

	Usage: asciiprint

Multi-Byte Characters

This validates that a string value contains one or more multibyte characters.
NOTE: if the string is blank, this validates as true.

	Usage: multibyte

Data URL

This validates that a string value contains a valid DataURI.
NOTE: this will also validate that the data portion is valid base64

	Usage: datauri

Latitude

This validates that a string value contains a valid latitude.

	Usage: latitude

Longitude

This validates that a string value contains a valid longitude.

	Usage: longitude

Social Security Number SSN

This validates that a string value contains a valid U.S. Social Security Number.

	Usage: ssn

Internet Protocol Address IP

This validates that a string value contains a valid IP Adress.

	Usage: ip

Internet Protocol Address IPv4

This validates that a string value contains a valid v4 IP Adress.

	Usage: ipv4

Internet Protocol Address IPv6

This validates that a string value contains a valid v6 IP Adress.

	Usage: ipv6

Classless Inter-Domain Routing CIDR

This validates that a string value contains a valid CIDR Adress.

	Usage: cidr

Classless Inter-Domain Routing CIDRv4

This validates that a string value contains a valid v4 CIDR Adress.

	Usage: cidrv4

Classless Inter-Domain Routing CIDRv6

This validates that a string value contains a valid v6 CIDR Adress.

	Usage: cidrv6

Transmission Control Protocol Address TCP

This validates that a string value contains a valid resolvable TCP Adress.

	Usage: tcp_addr

Transmission Control Protocol Address TCPv4

This validates that a string value contains a valid resolvable v4 TCP Adress.

	Usage: tcp4_addr

Transmission Control Protocol Address TCPv6

This validates that a string value contains a valid resolvable v6 TCP Adress.

	Usage: tcp6_addr

User Datagram Protocol Address UDP

This validates that a string value contains a valid resolvable UDP Adress.

	Usage: udp_addr

User Datagram Protocol Address UDPv4

This validates that a string value contains a valid resolvable v4 UDP Adress.

	Usage: udp4_addr

User Datagram Protocol Address UDPv6

This validates that a string value contains a valid resolvable v6 UDP Adress.

	Usage: udp6_addr

Internet Protocol Address IP

This validates that a string value contains a valid resolvable IP Adress.

	Usage: ip_addr

Internet Protocol Address IPv4

This validates that a string value contains a valid resolvable v4 IP Adress.

	Usage: ip4_addr

Internet Protocol Address IPv6

This validates that a string value contains a valid resolvable v6 IP Adress.

	Usage: ip6_addr

Unix domain socket end point Address

This validates that a string value contains a valid Unix Adress.

	Usage: unix_addr

Media Access Control Address MAC

This validates that a string value contains a valid MAC Adress.

	Usage: mac

Note: See Go's ParseMAC for accepted formats and types:

	http://golang.org/src/net/mac.go?s=866:918#L29

Alias Validators and Tags

NOTE: When returning an error, the tag returned in "FieldError" will be
the alias tag unless the dive tag is part of the alias. Everything after the
dive tag is not reported as the alias tag. Also, the "ActualTag" in the before
case will be the actual tag within the alias that failed.

Here is a list of the current built in alias tags:

	"iscolor"
		alias is "hexcolor|rgb|rgba|hsl|hsla" (Usage: iscolor)

Validator notes:

	regex
		a regex validator won't be added because commas and = signs can be part
		of a regex which conflict with the validation definitions. Although
		workarounds can be made, they take away from using pure regex's.
		Furthermore it's quick and dirty but the regex's become harder to
		maintain and are not reusable, so it's as much a programming philosiphy
		as anything.

		In place of this new validator functions should be created; a regex can
		be used within the validator function and even be precompiled for better
		efficiency within regexes.go.

		And the best reason, you can submit a pull request and we can keep on
		adding to the validation library of this package!

Panics

This package panics when bad input is provided, this is by design, bad code like
that should not make it to production.

	type Test struct {
		TestField string `validate:"nonexistantfunction=1"`
	}

	t := &Test{
		TestField: "Test"
	}

	validate.Struct(t) // this will panic
*/
package validator
