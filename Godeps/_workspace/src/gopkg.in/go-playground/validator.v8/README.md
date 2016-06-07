Package validator
================
<img align="right" src="https://raw.githubusercontent.com/go-playground/validator/v8/logo.png">
[![Join the chat at https://gitter.im/bluesuncorp/validator](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/go-playground/validator?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
![Project status](https://img.shields.io/badge/version-8.17.1-green.svg)
[![Build Status](https://semaphoreci.com/api/v1/projects/ec20115f-ef1b-4c7d-9393-cc76aba74eb4/530054/badge.svg)](https://semaphoreci.com/joeybloggs/validator)
[![Coverage Status](https://coveralls.io/repos/go-playground/validator/badge.svg?branch=v8&service=github)](https://coveralls.io/github/go-playground/validator?branch=v8)
[![Go Report Card](http://goreportcard.com/badge/go-playground/validator)](http://goreportcard.com/report/go-playground/validator)
[![GoDoc](https://godoc.org/gopkg.in/go-playground/validator.v8?status.svg)](https://godoc.org/gopkg.in/go-playground/validator.v8)
![License](https://img.shields.io/dub/l/vibe-d.svg)

Package validator implements value validations for structs and individual fields based on tags.

It has the following **unique** features:

-   Cross Field and Cross Struct validations by using validation tags or custom validators.  
-   Slice, Array and Map diving, which allows any or all levels of a multidimensional field to be validated.  
-   Handles type interface by determining it's underlying type prior to validation.
-   Handles custom field types such as sql driver Valuer see [Valuer](https://golang.org/src/database/sql/driver/types.go?s=1210:1293#L29)
-   Alias validation tags, which allows for mapping of several validations to a single tag for easier defining of validations on structs
-   Extraction of custom defined Field Name e.g. can specify to extract the JSON name while validating and have it available in the resulting FieldError

Installation
------------

Use go get.

	go get gopkg.in/go-playground/validator.v8

or to update

	go get -u gopkg.in/go-playground/validator.v8

Then import the validator package into your own code.

	import "gopkg.in/go-playground/validator.v8"

Error Return Value
-------

Validation functions return type error

They return type error to avoid the issue discussed in the following, where err is always != nil:

* http://stackoverflow.com/a/29138676/3158232
* https://github.com/go-playground/validator/issues/134

validator only returns nil or ValidationErrors as type error; so in you code all you need to do
is check if the error returned is not nil, and if it's not type cast it to type ValidationErrors
like so:

```go
err := validate.Struct(mystruct)
validationErrors := err.(validator.ValidationErrors)
 ```

Usage and documentation
------

Please see http://godoc.org/gopkg.in/go-playground/validator.v8 for detailed usage docs.

##### Examples:

Struct & Field validation
```go
package main

import (
	"fmt"

	"gopkg.in/go-playground/validator.v8"
)

// User contains user information
type User struct {
	FirstName      string     `validate:"required"`
	LastName       string     `validate:"required"`
	Age            uint8      `validate:"gte=0,lte=130"`
	Email          string     `validate:"required,email"`
	FavouriteColor string     `validate:"hexcolor|rgb|rgba"`
	Addresses      []*Address `validate:"required,dive,required"` // a person can have a home and cottage...
}

// Address houses a users address information
type Address struct {
	Street string `validate:"required"`
	City   string `validate:"required"`
	Planet string `validate:"required"`
	Phone  string `validate:"required"`
}

var validate *validator.Validate

func main() {

	config := &validator.Config{TagName: "validate"}

	validate = validator.New(config)

	validateStruct()
	validateField()
}

func validateStruct() {

	address := &Address{
		Street: "Eavesdown Docks",
		Planet: "Persphone",
		Phone:  "none",
	}

	user := &User{
		FirstName:      "Badger",
		LastName:       "Smith",
		Age:            135,
		Email:          "Badger.Smith@gmail.com",
		FavouriteColor: "#000",
		Addresses:      []*Address{address},
	}

	// returns nil or ValidationErrors ( map[string]*FieldError )
	errs := validate.Struct(user)

	if errs != nil {

		fmt.Println(errs) // output: Key: "User.Age" Error:Field validation for "Age" failed on the "lte" tag
		//	                         Key: "User.Addresses[0].City" Error:Field validation for "City" failed on the "required" tag
		err := errs.(validator.ValidationErrors)["User.Addresses[0].City"]
		fmt.Println(err.Field) // output: City
		fmt.Println(err.Tag)   // output: required
		fmt.Println(err.Kind)  // output: string
		fmt.Println(err.Type)  // output: string
		fmt.Println(err.Param) // output:
		fmt.Println(err.Value) // output:

		// from here you can create your own error messages in whatever language you wish
		return
	}

	// save user to database
}

func validateField() {
	myEmail := "joeybloggs.gmail.com"

	errs := validate.Field(myEmail, "required,email")

	if errs != nil {
		fmt.Println(errs) // output: Key: "" Error:Field validation for "" failed on the "email" tag
		return
	}

	// email ok, move on
}
```

Custom Field Type
```go
package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"

	"gopkg.in/go-playground/validator.v8"
)

// DbBackedUser User struct
type DbBackedUser struct {
	Name sql.NullString `validate:"required"`
	Age  sql.NullInt64  `validate:"required"`
}

func main() {

	config := &validator.Config{TagName: "validate"}

	validate := validator.New(config)

	// register all sql.Null* types to use the ValidateValuer CustomTypeFunc
	validate.RegisterCustomTypeFunc(ValidateValuer, sql.NullString{}, sql.NullInt64{}, sql.NullBool{}, sql.NullFloat64{})

	x := DbBackedUser{Name: sql.NullString{String: "", Valid: true}, Age: sql.NullInt64{Int64: 0, Valid: false}}
	errs := validate.Struct(x)

	if len(errs.(validator.ValidationErrors)) > 0 {
		fmt.Printf("Errs:\n%+v\n", errs)
	}
}

// ValidateValuer implements validator.CustomTypeFunc
func ValidateValuer(field reflect.Value) interface{} {
	if valuer, ok := field.Interface().(driver.Valuer); ok {
		val, err := valuer.Value()
		if err == nil {
			return val
		}
		// handle the error how you want
	}
	return nil
}
```

Struct Level Validation
```go
package main

import (
	"fmt"
	"reflect"

	"gopkg.in/go-playground/validator.v8"
)

// User contains user information
type User struct {
	FirstName      string     `json:"fname"`
	LastName       string     `json:"lname"`
	Age            uint8      `validate:"gte=0,lte=130"`
	Email          string     `validate:"required,email"`
	FavouriteColor string     `validate:"hexcolor|rgb|rgba"`
	Addresses      []*Address `validate:"required,dive,required"` // a person can have a home and cottage...
}

// Address houses a users address information
type Address struct {
	Street string `validate:"required"`
	City   string `validate:"required"`
	Planet string `validate:"required"`
	Phone  string `validate:"required"`
}

var validate *validator.Validate

func main() {

	config := &validator.Config{TagName: "validate"}

	validate = validator.New(config)
	validate.RegisterStructValidation(UserStructLevelValidation, User{})

	validateStruct()
}

// UserStructLevelValidation contains custom struct level validations that don't always
// make sense at the field validation level. For Example this function validates that either
// FirstName or LastName exist; could have done that with a custom field validation but then
// would have had to add it to both fields duplicating the logic + overhead, this way it's
// only validated once.
//
// NOTE: you may ask why wouldn't I just do this outside of validator, because doing this way
// hooks right into validator and you can combine with validation tags and still have a
// common error output format.
func UserStructLevelValidation(v *validator.Validate, structLevel *validator.StructLevel) {

	user := structLevel.CurrentStruct.Interface().(User)

	if len(user.FirstName) == 0 && len(user.LastName) == 0 {
		structLevel.ReportError(reflect.ValueOf(user.FirstName), "FirstName", "fname", "fnameorlname")
		structLevel.ReportError(reflect.ValueOf(user.LastName), "LastName", "lname", "fnameorlname")
	}

	// plus can to more, even with different tag than "fnameorlname"
}

func validateStruct() {

	address := &Address{
		Street: "Eavesdown Docks",
		Planet: "Persphone",
		Phone:  "none",
		City:   "Unknown",
	}

	user := &User{
		FirstName:      "",
		LastName:       "",
		Age:            45,
		Email:          "Badger.Smith@gmail.com",
		FavouriteColor: "#000",
		Addresses:      []*Address{address},
	}

	// returns nil or ValidationErrors ( map[string]*FieldError )
	errs := validate.Struct(user)

	if errs != nil {

		fmt.Println(errs) // output: Key: 'User.LastName' Error:Field validation for 'LastName' failed on the 'fnameorlname' tag
		//	                         Key: 'User.FirstName' Error:Field validation for 'FirstName' failed on the 'fnameorlname' tag
		err := errs.(validator.ValidationErrors)["User.FirstName"]
		fmt.Println(err.Field) // output: FirstName
		fmt.Println(err.Tag)   // output: fnameorlname
		fmt.Println(err.Kind)  // output: string
		fmt.Println(err.Type)  // output: string
		fmt.Println(err.Param) // output:
		fmt.Println(err.Value) // output:

		// from here you can create your own error messages in whatever language you wish
		return
	}

	// save user to database
}
```

Benchmarks
------
###### Run on MacBook Pro (Retina, 15-inch, Late 2013) 2.6 GHz Intel Core i7 16 GB 1600 MHz DDR3 using Go version go1.5.3 darwin/amd64
```go
go test -cpu=4 -bench=. -benchmem=true
PASS
BenchmarkFieldSuccess-4                            	10000000	       167 ns/op	       0 B/op	       0 allocs/op
BenchmarkFieldFailure-4                            	 2000000	       701 ns/op	     432 B/op	       4 allocs/op
BenchmarkFieldDiveSuccess-4                        	  500000	      2937 ns/op	     480 B/op	      27 allocs/op
BenchmarkFieldDiveFailure-4                        	  500000	      3536 ns/op	     912 B/op	      31 allocs/op
BenchmarkFieldCustomTypeSuccess-4                  	 5000000	       341 ns/op	      32 B/op	       2 allocs/op
BenchmarkFieldCustomTypeFailure-4                  	 2000000	       679 ns/op	     432 B/op	       4 allocs/op
BenchmarkFieldOrTagSuccess-4                       	 1000000	      1157 ns/op	      16 B/op	       1 allocs/op
BenchmarkFieldOrTagFailure-4                       	 1000000	      1109 ns/op	     464 B/op	       6 allocs/op
BenchmarkStructLevelValidationSuccess-4            	 2000000	       694 ns/op	     176 B/op	       6 allocs/op
BenchmarkStructLevelValidationFailure-4            	 1000000	      1311 ns/op	     640 B/op	      11 allocs/op
BenchmarkStructSimpleCustomTypeSuccess-4           	 2000000	       894 ns/op	      80 B/op	       5 allocs/op
BenchmarkStructSimpleCustomTypeFailure-4           	 1000000	      1496 ns/op	     688 B/op	      11 allocs/op
BenchmarkStructPartialSuccess-4                    	 1000000	      1229 ns/op	     384 B/op	      10 allocs/op
BenchmarkStructPartialFailure-4                    	 1000000	      1838 ns/op	     832 B/op	      15 allocs/op
BenchmarkStructExceptSuccess-4                     	 2000000	       961 ns/op	     336 B/op	       7 allocs/op
BenchmarkStructExceptFailure-4                     	 1000000	      1218 ns/op	     384 B/op	      10 allocs/op
BenchmarkStructSimpleCrossFieldSuccess-4           	 2000000	       954 ns/op	     128 B/op	       6 allocs/op
BenchmarkStructSimpleCrossFieldFailure-4           	 1000000	      1569 ns/op	     592 B/op	      11 allocs/op
BenchmarkStructSimpleCrossStructCrossFieldSuccess-4	 1000000	      1588 ns/op	     192 B/op	      10 allocs/op
BenchmarkStructSimpleCrossStructCrossFieldFailure-4	 1000000	      2217 ns/op	     656 B/op	      15 allocs/op
BenchmarkStructSimpleSuccess-4                     	 2000000	       925 ns/op	      48 B/op	       3 allocs/op
BenchmarkStructSimpleFailure-4                     	 1000000	      1650 ns/op	     688 B/op	      11 allocs/op
BenchmarkStructSimpleSuccessParallel-4             	 5000000	       261 ns/op	      48 B/op	       3 allocs/op
BenchmarkStructSimpleFailureParallel-4             	 2000000	       758 ns/op	     688 B/op	      11 allocs/op
BenchmarkStructComplexSuccess-4                    	  300000	      5868 ns/op	     544 B/op	      32 allocs/op
BenchmarkStructComplexFailure-4                    	  200000	     10767 ns/op	    3912 B/op	      77 allocs/op
BenchmarkStructComplexSuccessParallel-4            	 1000000	      1559 ns/op	     544 B/op	      32 allocs/op
BenchmarkStructComplexFailureParallel-4            	  500000	      3747 ns/op	    3912 B/op	      77 allocs
```

Complimentary Software
----------------------

Here is a list of software that compliments using this library either pre or post validation.

* [Gorilla Schema](https://github.com/gorilla/schema) - Package gorilla/schema fills a struct with form values.
* [Conform](https://github.com/leebenson/conform) - Trims, sanitizes & scrubs data based on struct tags.

How to Contribute
------

There will always be a development branch for each version i.e. `v1-development`. In order to contribute, 
please make your pull requests against those branches.

If the changes being proposed or requested are breaking changes, please create an issue, for discussion
or create a pull request against the highest development branch for example this package has a
v1 and v1-development branch however, there will also be a v2-development branch even though v2 doesn't exist yet.

I strongly encourage everyone whom creates a custom validation function to contribute them and
help make this package even better.

License
------
Distributed under MIT License, please see license file in code for more details.
