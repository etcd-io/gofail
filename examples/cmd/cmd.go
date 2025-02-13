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

package main

import (
	"log"
	"time"

	"go.etcd.io/gofail/examples"
)

/*
GOFAIL_HTTP=:22381 go run cmd.go

curl -L http://localhost:22381

curl \
  -L http://localhost:22381/ExampleLabels \
  -X PUT -d'return'

curl \
  -L http://localhost:22381/ExampleLabels \
  -X DELETE
*/

func main() {
	for {
		log.Println(examples.ExampleFunc())
		log.Println(examples.ExampleOneLineFunc())
		log.Println(examples.ExampleLabelsFunc())
		time.Sleep(time.Second)
	}
}
