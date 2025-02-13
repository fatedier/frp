// Copyright 2024 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"

	"github.com/fatedier/frp/pkg/util/util"
)

type NumberPair struct {
	First  int64
	Second int64
}

func parseNumberRangePair(firstRangeStr, secondRangeStr string) ([]NumberPair, error) {
	firstRangeNumbers, err := util.ParseRangeNumbers(firstRangeStr)
	if err != nil {
		return nil, err
	}
	secondRangeNumbers, err := util.ParseRangeNumbers(secondRangeStr)
	if err != nil {
		return nil, err
	}
	if len(firstRangeNumbers) != len(secondRangeNumbers) {
		return nil, fmt.Errorf("first and second range numbers are not in pairs")
	}
	pairs := make([]NumberPair, 0, len(firstRangeNumbers))
	for i := 0; i < len(firstRangeNumbers); i++ {
		pairs = append(pairs, NumberPair{
			First:  firstRangeNumbers[i],
			Second: secondRangeNumbers[i],
		})
	}
	return pairs, nil
}

func parseNumberRange(firstRangeStr string) ([]int64, error) {
	return util.ParseRangeNumbers(firstRangeStr)
}
