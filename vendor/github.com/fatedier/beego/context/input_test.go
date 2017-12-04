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

package context

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestBind(t *testing.T) {
	type testItem struct {
		field string
		empty interface{}
		want  interface{}
	}
	type Human struct {
		ID   int
		Nick string
		Pwd  string
		Ms   bool
	}

	cases := []struct {
		request string
		valueGp []testItem
	}{
		{"/?p=str", []testItem{{"p", interface{}(""), interface{}("str")}}},

		{"/?p=", []testItem{{"p", "", ""}}},
		{"/?p=str", []testItem{{"p", "", "str"}}},

		{"/?p=123", []testItem{{"p", 0, 123}}},
		{"/?p=123", []testItem{{"p", uint(0), uint(123)}}},

		{"/?p=1.0", []testItem{{"p", 0.0, 1.0}}},
		{"/?p=1", []testItem{{"p", false, true}}},

		{"/?p=true", []testItem{{"p", false, true}}},
		{"/?p=ON", []testItem{{"p", false, true}}},
		{"/?p=on", []testItem{{"p", false, true}}},
		{"/?p=1", []testItem{{"p", false, true}}},
		{"/?p=2", []testItem{{"p", false, false}}},
		{"/?p=false", []testItem{{"p", false, false}}},

		{"/?p[a]=1&p[b]=2&p[c]=3", []testItem{{"p", map[string]int{}, map[string]int{"a": 1, "b": 2, "c": 3}}}},
		{"/?p[a]=v1&p[b]=v2&p[c]=v3", []testItem{{"p", map[string]string{}, map[string]string{"a": "v1", "b": "v2", "c": "v3"}}}},

		{"/?p[]=8&p[]=9&p[]=10", []testItem{{"p", []int{}, []int{8, 9, 10}}}},
		{"/?p[0]=8&p[1]=9&p[2]=10", []testItem{{"p", []int{}, []int{8, 9, 10}}}},
		{"/?p[0]=8&p[1]=9&p[2]=10&p[5]=14", []testItem{{"p", []int{}, []int{8, 9, 10, 0, 0, 14}}}},
		{"/?p[0]=8.0&p[1]=9.0&p[2]=10.0", []testItem{{"p", []float64{}, []float64{8.0, 9.0, 10.0}}}},

		{"/?p[]=10&p[]=9&p[]=8", []testItem{{"p", []string{}, []string{"10", "9", "8"}}}},
		{"/?p[0]=8&p[1]=9&p[2]=10", []testItem{{"p", []string{}, []string{"8", "9", "10"}}}},

		{"/?p[0]=true&p[1]=false&p[2]=true&p[5]=1&p[6]=ON&p[7]=other", []testItem{{"p", []bool{}, []bool{true, false, true, false, false, true, true, false}}}},

		{"/?human.Nick=astaxie", []testItem{{"human", Human{}, Human{Nick: "astaxie"}}}},
		{"/?human.ID=888&human.Nick=astaxie&human.Ms=true&human[Pwd]=pass", []testItem{{"human", Human{}, Human{ID: 888, Nick: "astaxie", Ms: true, Pwd: "pass"}}}},
		{"/?human[0].ID=888&human[0].Nick=astaxie&human[0].Ms=true&human[0][Pwd]=pass01&human[1].ID=999&human[1].Nick=ysqi&human[1].Ms=On&human[1].Pwd=pass02",
			[]testItem{{"human", []Human{}, []Human{
				Human{ID: 888, Nick: "astaxie", Ms: true, Pwd: "pass01"},
				Human{ID: 999, Nick: "ysqi", Ms: true, Pwd: "pass02"},
			}}}},

		{
			"/?id=123&isok=true&ft=1.2&ol[0]=1&ol[1]=2&ul[]=str&ul[]=array&human.Nick=astaxie",
			[]testItem{
				{"id", 0, 123},
				{"isok", false, true},
				{"ft", 0.0, 1.2},
				{"ol", []int{}, []int{1, 2}},
				{"ul", []string{}, []string{"str", "array"}},
				{"human", Human{}, Human{Nick: "astaxie"}},
			},
		},
	}
	for _, c := range cases {
		r, _ := http.NewRequest("GET", c.request, nil)
		beegoInput := NewInput()
		beegoInput.Context = NewContext()
		beegoInput.Context.Reset(httptest.NewRecorder(), r)

		for _, item := range c.valueGp {
			got := item.empty
			err := beegoInput.Bind(&got, item.field)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, item.want) {
				t.Fatalf("Bind %q error,should be:\n%#v \ngot:\n%#v", item.field, item.want, got)
			}
		}

	}
}

func TestSubDomain(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://www.example.com/?id=123&isok=true&ft=1.2&ol[0]=1&ol[1]=2&ul[]=str&ul[]=array&user.Name=astaxie", nil)
	beegoInput := NewInput()
	beegoInput.Context = NewContext()
	beegoInput.Context.Reset(httptest.NewRecorder(), r)

	subdomain := beegoInput.SubDomains()
	if subdomain != "www" {
		t.Fatal("Subdomain parse error, got" + subdomain)
	}

	r, _ = http.NewRequest("GET", "http://localhost/", nil)
	beegoInput.Context.Request = r
	if beegoInput.SubDomains() != "" {
		t.Fatal("Subdomain parse error, should be empty, got " + beegoInput.SubDomains())
	}

	r, _ = http.NewRequest("GET", "http://aa.bb.example.com/", nil)
	beegoInput.Context.Request = r
	if beegoInput.SubDomains() != "aa.bb" {
		t.Fatal("Subdomain parse error, got " + beegoInput.SubDomains())
	}

	/* TODO Fix this
	r, _ = http.NewRequest("GET", "http://127.0.0.1/", nil)
	beegoInput.Context.Request = r
	if beegoInput.SubDomains() != "" {
		t.Fatal("Subdomain parse error, got " + beegoInput.SubDomains())
	}
	*/

	r, _ = http.NewRequest("GET", "http://example.com/", nil)
	beegoInput.Context.Request = r
	if beegoInput.SubDomains() != "" {
		t.Fatal("Subdomain parse error, got " + beegoInput.SubDomains())
	}

	r, _ = http.NewRequest("GET", "http://aa.bb.cc.dd.example.com/", nil)
	beegoInput.Context.Request = r
	if beegoInput.SubDomains() != "aa.bb.cc.dd" {
		t.Fatal("Subdomain parse error, got " + beegoInput.SubDomains())
	}
}

func TestParams(t *testing.T) {
	inp := NewInput()

	inp.SetParam("p1", "val1_ver1")
	inp.SetParam("p2", "val2_ver1")
	inp.SetParam("p3", "val3_ver1")
	if l := inp.ParamsLen(); l != 3 {
		t.Fatalf("Input.ParamsLen wrong value: %d, expected %d", l, 3)
	}

	if val := inp.Param("p1"); val != "val1_ver1" {
		t.Fatalf("Input.Param wrong value: %s, expected %s", val, "val1_ver1")
	}
	if val := inp.Param("p3"); val != "val3_ver1" {
		t.Fatalf("Input.Param wrong value: %s, expected %s", val, "val3_ver1")
	}
	vals := inp.Params()
	expected := map[string]string{
		"p1": "val1_ver1",
		"p2": "val2_ver1",
		"p3": "val3_ver1",
	}
	if !reflect.DeepEqual(vals, expected) {
		t.Fatalf("Input.Params wrong value: %s, expected %s", vals, expected)
	}

	// overwriting existing params
	inp.SetParam("p1", "val1_ver2")
	inp.SetParam("p2", "val2_ver2")
	expected = map[string]string{
		"p1": "val1_ver2",
		"p2": "val2_ver2",
		"p3": "val3_ver1",
	}
	vals = inp.Params()
	if !reflect.DeepEqual(vals, expected) {
		t.Fatalf("Input.Params wrong value: %s, expected %s", vals, expected)
	}

	if l := inp.ParamsLen(); l != 3 {
		t.Fatalf("Input.ParamsLen wrong value: %d, expected %d", l, 3)
	}

	if val := inp.Param("p1"); val != "val1_ver2" {
		t.Fatalf("Input.Param wrong value: %s, expected %s", val, "val1_ver2")
	}

	if val := inp.Param("p2"); val != "val2_ver2" {
		t.Fatalf("Input.Param wrong value: %s, expected %s", val, "val1_ver2")
	}

}
