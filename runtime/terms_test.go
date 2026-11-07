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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err)
		for _, w := range tt.weval {
			v := ter.eval()
			if v == nil && w == "" {
				continue
			}
			require.Equalf(t, v.(string), w, "got %q, expected %q", v, w)
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
		require.NoError(t, err)
		v := ter.eval()
		if v == nil && tt.weval == nil {
			continue
		}
		require.Truef(t, reflect.DeepEqual(v, tt.weval), "got %v, expected %v", v, tt.weval)
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
		require.NoError(t, err)
		for i := 0; i < tt.runAfterEnabling; i++ {
			_ = ter.eval()
		}

		assert.Equalf(t, tt.wantCount, ter.counter, "counter is not properly incremented, got: %d, want: %d", ter.counter, tt.wantCount)
	}
}

func TestTermsQuotedValue(t *testing.T) {
	tests := []struct {
		desc  string
		weval interface{}
	}{
		{`return("abc")`, "abc"},
		{`return("a\"b")`, `a"b`},
		{`return("a\nb")`, "a\nb"},
		{`return("a\\b")`, `a\b`},
		{`return("tab\there")`, "tab\there"},
		{`return("")`, ""},
	}
	for _, tt := range tests {
		ter, err := newTerms("test", tt.desc)
		require.NoErrorf(t, err, "could not parse %s", tt.desc)
		require.Equalf(t, tt.desc, ter.String(), "term description round trip for %s", tt.desc)
		require.Equalf(t, tt.weval, ter.eval(), "value for %s", tt.desc)
	}
}

func TestTermsQuotedValueInChain(t *testing.T) {
	ter, err := newTerms("test", `1*return("a\"b")->return("x")`)
	require.NoError(t, err)
	assert.Equal(t, `a"b`, ter.eval())
	assert.Equal(t, "x", ter.eval())
}

func TestTermsUnterminatedValueIsRejected(t *testing.T) {
	for _, desc := range []string{
		`return("abc"`,
		`return("abc`,
		`return("`,
		`return(1`,
		`return(true`,
	} {
		_, err := newTerms("test", desc)
		assert.Errorf(t, err, "expected %s to be rejected, not to panic", desc)
	}
}

func TestTermsIntKeepsLeadingZeroes(t *testing.T) {
	ter, err := newTerms("test", `return(0012)`)
	require.NoError(t, err)
	assert.Equal(t, `return(0012)`, ter.String())
	assert.Equal(t, 12, ter.eval())
}
