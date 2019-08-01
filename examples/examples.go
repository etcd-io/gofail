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

package examples

func ExampleFunc() string {
	// gofail: var ExampleString string
	// return ExampleString
	return "example"
}

func ExampleOneLineFunc() string {
	// gofail: var ExampleOneLine struct{}
	return "abc"
}

func ExampleLabelsFunc() (s string) {
	i := 0
	// gofail: myLabel:
	for i < 5 {
		s = s + "i"
		i++
		for j := 0; j < 5; j++ {
			s = s + "j"
			// gofail: var ExampleLabels struct{}
			// continue myLabel
		}
	}
	return s
}

func ExampleLabelsGoFunc() (s string) {
	i := 0
	// gofail: myLabel:
	for i < 5 {
		s = s + "i"
		i++
		for j := 0; j < 5; j++ {
			s = s + "j"
			// gofail-go: var ExampleLabelsGo struct{}
			// continue myLabel
		}
	}
	return s
}
