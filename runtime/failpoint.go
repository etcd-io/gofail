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
	"context"
	"fmt"
	"sync"
)

type Failpoint struct {
	cmu    sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	donec  chan struct{}
	// gofail-go failpoint's release process will be blocked until it's
	// deleted/disabled by users. The purpose is to make sure users can't
	// update a gofail-go failpoint until it's deleted/disabled.
	releasing bool
	released  bool

	mu sync.RWMutex
	t  *terms
}

func NewFailpoint(pkg, name string, goFailGo bool) *Failpoint {
	fp := &Failpoint{}
	if goFailGo {
		fp.ctx, fp.cancel = context.WithCancel(context.Background())
		fp.donec = make(chan struct{})
	}
	register(pkg+"/"+name, fp)
	return fp
}

// Acquire gets evalutes the failpoint terms; if the failpoint
// is active, it will return a value. Otherwise, returns a non-nil error.
func (fp *Failpoint) Acquire() (interface{}, error) {
	fp.mu.RLock()
	if fp.t == nil {
		fp.mu.RUnlock()
		return nil, ErrDisabled
	}
	v, err := fp.t.eval()
	if v == nil {
		err = ErrDisabled
	}
	if err != nil {
		fp.mu.RUnlock()
	}
	return v, err
}

// Release is called when the failpoint exists.
func (fp *Failpoint) Release() {
	fp.cmu.Lock()
	fp.releasing = true
	fp.cmu.Unlock()

	fp.cmu.RLock()
	ctx := fp.ctx
	donec := fp.donec
	released := fp.released
	fp.cmu.RUnlock()
	if ctx != nil && !released {
		<-ctx.Done()
		select {
		case <-donec:
		default:
			close(donec)
		}
	}

	fp.cmu.Lock()
	fp.releasing = false
	fp.cmu.Unlock()

	fp.mu.RUnlock()
}

// BadType is called when the failpoint evaluates to the wrong type.
func (fp *Failpoint) BadType(v interface{}, t string) {
	fmt.Printf("failpoint: %q got value %v of type \"%T\" but expected type %q\n", fp.t.fpath, v, v, t)
}
