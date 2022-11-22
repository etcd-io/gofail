// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFailpoints(t *testing.T) {
	testCases := []struct {
		name           string
		fps            string
		expectedFpsMap map[string]string
	}{
		{
			name:           "only one valid failpoint",
			fps:            "failpoint1=print",
			expectedFpsMap: map[string]string{"failpoint1": "print"},
		},
		{
			name:           "only one invalid failpoint",
			fps:            "failpoint1",
			expectedFpsMap: nil,
		},
		{
			name:           "multiple valid failpoints",
			fps:            "failpoint1=print;failpoint2=sleep(10)",
			expectedFpsMap: map[string]string{"failpoint1": "print", "failpoint2": "sleep(10)"},
		},
		{
			name:           "multiple invalid failpoints",
			fps:            "failpoint1=print_failpoint2=sleep(10)",
			expectedFpsMap: nil,
		},
		{
			name:           "partial valid failpoints",
			fps:            "failpoint1=print;failpoint2",
			expectedFpsMap: nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fpsMap, err := parseFailpoints(tc.fps)

			// When tc.expectedFpsMap is nil, then we expect `parseFailpoints` returns error.
			require.Equal(t, tc.expectedFpsMap == nil, err != nil, "Unexpected result, tc.expectedFpsMap: %v, err: %v", tc.expectedFpsMap, err)

			require.Equal(t, tc.expectedFpsMap, fpsMap, "Unexpected result, expected: %v, got: %v", tc.expectedFpsMap, fpsMap)
		})
	}
}
