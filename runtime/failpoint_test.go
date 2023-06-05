// Copyright 2022 The etcd Authors
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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFailpointCreateAndAcquire(t *testing.T) {
	name := "failpoint"
	envTerms = map[string]string{name: "return(1)"}
	defer clearGlobalVars()

	fp1 := NewFailpoint("failpoint")

	assert.NotNil(t, fp1.t)
	assert.Equal(t, envTerms[name], fp1.t.desc)

	v, err := fp1.Acquire()
	assert.Nil(t, err)

	intv := v.(int)
	assert.Equal(t, 1, intv)
}

func TestSameFailpointCreateTwice(t *testing.T) {
	name := "failpoint"
	envTerms = map[string]string{name: "print"}
	defer clearGlobalVars()

	NewFailpoint("failpoint")
	assert.Panics(t, func() { NewFailpoint("failpoint") })
}

// clearGlobalVars will unset runtime package global variables
// note: doesn't work if tests are run in parallel
func clearGlobalVars() {
	envTerms = make(map[string]string)
	failpoints = make(map[string]*Failpoint)
}
