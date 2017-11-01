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

package toolbox

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStatics(t *testing.T) {
	StatisticsMap.AddStatistics("POST", "/api/user", "&admin.user", time.Duration(2000))
	StatisticsMap.AddStatistics("POST", "/api/user", "&admin.user", time.Duration(120000))
	StatisticsMap.AddStatistics("GET", "/api/user", "&admin.user", time.Duration(13000))
	StatisticsMap.AddStatistics("POST", "/api/admin", "&admin.user", time.Duration(14000))
	StatisticsMap.AddStatistics("POST", "/api/user/astaxie", "&admin.user", time.Duration(12000))
	StatisticsMap.AddStatistics("POST", "/api/user/xiemengjun", "&admin.user", time.Duration(13000))
	StatisticsMap.AddStatistics("DELETE", "/api/user", "&admin.user", time.Duration(1400))
	t.Log(StatisticsMap.GetMap())

	data := StatisticsMap.GetMapData()
	b, err := json.Marshal(data)
	if err != nil {
		t.Errorf(err.Error())
	}

	t.Log(string(b))
}
