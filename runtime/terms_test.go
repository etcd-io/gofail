// Copyright 2016 CoreOS, Inc.
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

package runtime

import (
	"reflect"
	"testing"
)

func TestTermsString(t *testing.T) {
	tests := []struct {
		desc  string
		weval []string
	}{
		{`off`, []string{""}},
		{`2*return("abc")`, []string{"abc", "abc", ""}},
		{`2*return("abc")->1*return("def")`, []string{"abc", "abc", "def", ""}},
		{`1*return("abc")->return("def")`, []string{"abc", "def", "def"}},
	}
	for _, tt := range tests {
		ter, err := newTerms("test", tt.desc)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range tt.weval {
			v := ter.eval()
			if v == nil && w == "" {
				continue
			}
			if v.(string) != w {
				t.Fatalf("got %q, expected %q", v, w)
			}
		}
	}
}

func TestTermsTypes(t *testing.T) {
	tests := []struct {
		desc  string
		weval interface{}
	}{
		{`off`, nil},
		{`return("abc")`, "abc"},
		{`return(true)`, true},
		{`return(1)`, 1},
		{`return()`, struct{}{}},
	}
	for _, tt := range tests {
		ter, err := newTerms("test", tt.desc)
		if err != nil {
			t.Fatal(err)
		}
		v := ter.eval()
		if v == nil && tt.weval == nil {
			continue
		}
		if !reflect.DeepEqual(v, tt.weval) {
			t.Fatalf("got %v, expected %v", v, tt.weval)
		}
	}
}

func TestTermsCounter(t *testing.T) {
	tests := []struct {
		failpointTerm    string
		runAfterEnabling int
		wantCount        int
	}{
		{
			failpointTerm:    `10*sleep(10)->1*return("abc")`,
			runAfterEnabling: 12,
			// Note the chain of terms is allowed to be executed 11 times at most,
			// including 10 times for the first term `10*sleep(10)` and 1 time for
			// the second term `1*return("abc")`. So it's only evaluated 11 times
			// even it's triggered 12 times.
			wantCount: 11,
		},
		{
			failpointTerm:    `10*sleep(10)->1*return("abc")`,
			runAfterEnabling: 3,
			wantCount:        3,
		},
		{
			failpointTerm:    `10*sleep(10)->1*return("abc")`,
			runAfterEnabling: 0,
			wantCount:        0,
		},
	}
	for _, tt := range tests {
		ter, err := newTerms("test", tt.failpointTerm)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < tt.runAfterEnabling; i++ {
			_ = ter.eval()
		}

		if ter.counter != tt.wantCount {
			t.Errorf("counter is not properly incremented, got: %d, want: %d", ter.counter, tt.wantCount)
		}
	}
}
