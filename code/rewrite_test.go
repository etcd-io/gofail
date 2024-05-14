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

package code

import (
	"bytes"
	"strings"
	"testing"
)

var examples = []struct {
	code                  string
	expectedGeneratedCode string
	wfps                  int
}{
	{
		"func f() {\n\t// gofail: var Test int\n\t// fmt.Println(Test)\n}",
		"func f() {\n\tif vTest, __fpErr := __fp_Test.Acquire(); __fpErr == nil { Test, __fpTypeOK := vTest.(int); if !__fpTypeOK { goto __badTypeTest} \n\t\t fmt.Println(Test); goto __nomockTest; __badTypeTest: __fp_Test.BadType(vTest, \"int\"); __nomockTest: };\n}",
		1,
	},
	{
		"func f() {\n\t\t// gofail: var Test int\n\t\t// \tfmt.Println(Test)\n}",
		"func f() {\n\t\tif vTest, __fpErr := __fp_Test.Acquire(); __fpErr == nil { Test, __fpTypeOK := vTest.(int); if !__fpTypeOK { goto __badTypeTest} \n\t\t\t \tfmt.Println(Test); goto __nomockTest; __badTypeTest: __fp_Test.BadType(vTest, \"int\"); __nomockTest: };\n}",
		1,
	},
	{
		"func f() {\n// gofail: var Test int\n// \tfmt.Println(Test)\n}",
		"func f() {\nif vTest, __fpErr := __fp_Test.Acquire(); __fpErr == nil { Test, __fpTypeOK := vTest.(int); if !__fpTypeOK { goto __badTypeTest} \n\t \tfmt.Println(Test); goto __nomockTest; __badTypeTest: __fp_Test.BadType(vTest, \"int\"); __nomockTest: };\n}",
		1,
	},
	{
		"func f() {\n\t// gofail: var Test int\n\t// fmt.Println(Test)\n}\n",
		"func f() {\n\tif vTest, __fpErr := __fp_Test.Acquire(); __fpErr == nil { Test, __fpTypeOK := vTest.(int); if !__fpTypeOK { goto __badTypeTest} \n\t\t fmt.Println(Test); goto __nomockTest; __badTypeTest: __fp_Test.BadType(vTest, \"int\"); __nomockTest: };\n}\n",
		1},
	{
		"func f() {\n\t// gofail: var Test int\n\t// fmt.Println(Test)// return\n}\n",
		"func f() {\n\tif vTest, __fpErr := __fp_Test.Acquire(); __fpErr == nil { Test, __fpTypeOK := vTest.(int); if !__fpTypeOK { goto __badTypeTest} \n\t\t fmt.Println(Test)// return; goto __nomockTest; __badTypeTest: __fp_Test.BadType(vTest, \"int\"); __nomockTest: };\n}\n",
		1,
	},
	{
		"func f() {\n\t// gofail: var OneLineTest int\n}\n",
		"func f() {\n\tif vOneLineTest, __fpErr := __fp_OneLineTest.Acquire(); __fpErr == nil { _, __fpTypeOK := vOneLineTest.(int); if !__fpTypeOK { goto __badTypeOneLineTest} ; goto __nomockOneLineTest; __badTypeOneLineTest: __fp_OneLineTest.BadType(vOneLineTest, \"int\"); __nomockOneLineTest: };\n}\n",
		1,
	},
	{
		"func f() {\n\t// gofail: var Test int\n\t// fmt.Println(Test)\n\n\t// gofail: var Test2 int\n\t// fmt.Println(Test2)\n}\n",
		"func f() {\n\tif vTest, __fpErr := __fp_Test.Acquire(); __fpErr == nil { Test, __fpTypeOK := vTest.(int); if !__fpTypeOK { goto __badTypeTest} \n\t\t fmt.Println(Test); goto __nomockTest; __badTypeTest: __fp_Test.BadType(vTest, \"int\"); __nomockTest: };\n\n\tif vTest2, __fpErr := __fp_Test2.Acquire(); __fpErr == nil { Test2, __fpTypeOK := vTest2.(int); if !__fpTypeOK { goto __badTypeTest2} \n\t\t fmt.Println(Test2); goto __nomockTest2; __badTypeTest2: __fp_Test2.BadType(vTest2, \"int\"); __nomockTest2: };\n}\n",
		2,
	},
	{
		"func f() {\n\t// gofail: var NoTypeTest struct{}\n\t// fmt.Println(`hi`)\n}\n",
		"func f() {\n\tif vNoTypeTest, __fpErr := __fp_NoTypeTest.Acquire(); __fpErr == nil { _, __fpTypeOK := vNoTypeTest.(struct{}); if !__fpTypeOK { goto __badTypeNoTypeTest} \n\t\t fmt.Println(`hi`); goto __nomockNoTypeTest; __badTypeNoTypeTest: __fp_NoTypeTest.BadType(vNoTypeTest, \"struct{}\"); __nomockNoTypeTest: };\n}\n",
		1,
	},
	{
		"func f() {\n\t// gofail: var NoTypeTest struct{}\n}\n",
		"func f() {\n\tif vNoTypeTest, __fpErr := __fp_NoTypeTest.Acquire(); __fpErr == nil { _, __fpTypeOK := vNoTypeTest.(struct{}); if !__fpTypeOK { goto __badTypeNoTypeTest} ; goto __nomockNoTypeTest; __badTypeNoTypeTest: __fp_NoTypeTest.BadType(vNoTypeTest, \"struct{}\"); __nomockNoTypeTest: };\n}\n",
		1,
	},
	{
		"func f() {\n\t// gofail: var NoTypeTest struct{}\n\t// fmt.Println(`hi`)\n\t// fmt.Println(`bye`)\n}\n",
		"func f() {\n\tif vNoTypeTest, __fpErr := __fp_NoTypeTest.Acquire(); __fpErr == nil { _, __fpTypeOK := vNoTypeTest.(struct{}); if !__fpTypeOK { goto __badTypeNoTypeTest} \n\t\t fmt.Println(`hi`)\n\t\t fmt.Println(`bye`); goto __nomockNoTypeTest; __badTypeNoTypeTest: __fp_NoTypeTest.BadType(vNoTypeTest, \"struct{}\"); __nomockNoTypeTest: };\n}\n",
		1,
	},
	{
		`
func f() {
	// gofail: labelTest:
	for {
		if g() {
			// gofail: var testLabel struct{}
			// continue labelTest
			return
		}
	}
}
`,
		"\nfunc f() {\n\t/* gofail-label */ labelTest:\n\tfor {\n\t\tif g() {\n\t\t\tif vtestLabel, __fpErr := __fp_testLabel.Acquire(); __fpErr == nil { _, __fpTypeOK := vtestLabel.(struct{}); if !__fpTypeOK { goto __badTypetestLabel} \n\t\t\t\t continue labelTest; goto __nomocktestLabel; __badTypetestLabel: __fp_testLabel.BadType(vtestLabel, \"struct{}\"); __nomocktestLabel: };\n\t\t\treturn\n\t\t}\n\t}\n}\n",
		1,
	},
}

func TestToFailpoint(t *testing.T) {
	for i, ex := range examples {
		dst := bytes.NewBuffer(make([]byte, 0, 1024))
		src := strings.NewReader(ex.code)
		fps, err := ToFailpoints(dst, src)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}
		if len(fps) != ex.wfps {
			t.Fatalf("%d: got %d failpoints but expected %d", i, len(fps), ex.wfps)
		}
		dstOut := dst.String()
		if len(strings.Split(dstOut, "\n")) != len(strings.Split(ex.code, "\n")) {
			t.Fatalf("%d: bad line count %q", i, dstOut)
		}
		if ex.expectedGeneratedCode != dstOut {
			t.Fatalf("expected generated code and actual generated code differs:\nExpected:\n%q\n\nActual:\n%q", ex.expectedGeneratedCode, dstOut)
		}
	}
}

func TestToComment(t *testing.T) {
	for i, ex := range examples {
		dst := bytes.NewBuffer(make([]byte, 0, 1024))
		src := strings.NewReader(ex.code)
		_, err := ToFailpoints(dst, src)
		if err != nil {
			t.Fatalf("%d: %v", i, err)
		}

		src = strings.NewReader(dst.String())
		dst.Reset()
		fps, err := ToComments(dst, src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		plainCode := dst.String()

		if plainCode != ex.code {
			t.Fatalf("%d: non-preserving ToComments(); got %q, want %q", i, plainCode, ex.code)
		}
		if len(fps) != ex.wfps {
			t.Fatalf("%d: got %d failpoints but expected %d", i, len(fps), ex.wfps)
		}
	}
}
