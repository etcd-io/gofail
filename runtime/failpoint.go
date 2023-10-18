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
	"fmt"
)

type Failpoint struct {
	t *terms
}

func NewFailpoint(name string) *Failpoint {
	return register(name)
}

// Acquire gets evalutes the failpoint terms; if the failpoint
// is active, it will return a value. Otherwise, returns a non-nil error.
func (fp *Failpoint) Acquire() (interface{}, error) {
	failpointsMu.RLock()
	defer failpointsMu.RUnlock()

	if fp.t == nil {
		return nil, ErrDisabled
	}
	result := fp.t.eval()
	if result == nil {
		return nil, ErrDisabled
	}
	return result, nil
}

// BadType is called when the failpoint evaluates to the wrong type.
func (fp *Failpoint) BadType(v interface{}, t string) {
	fmt.Printf("failpoint: %q got value %v of type \"%T\" but expected type %q\n", fp.t.fpath, v, v, t)
}
