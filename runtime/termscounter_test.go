package runtime_test

import (
	"strings"
	"testing"
	"time"

	"go.etcd.io/gofail/runtime"
)

var __fp_ExampleString *runtime.Failpoint = runtime.NewFailpoint("runtime_test", "ExampleString") //nolint:stylecheck

func TestTermsCounter(t *testing.T) {
	testcases := []struct {
		name      string
		fp        string
		desc      string
		runbefore int
		runafter  int
		want      string
	}{
		{
			name:     "Terms limit Failpoint",
			fp:       "runtime_test/ExampleString",
			desc:     `10*sleep(10)->1*return("abc")`,
			runafter: 12,
			want:     "11",
		},
		{
			name:      "Inbetween Enabling Failpoint",
			fp:        "runtime_test/ExampleString",
			desc:      `10*sleep(10)->1*return("abc")`,
			runbefore: 2,
			runafter:  3,
			want:      "3",
		},
		{
			name:      "Before Enabling Failpoint",
			fp:        "runtime_test/ExampleString",
			desc:      `10*sleep(10)->1*return("abc")`,
			runbefore: 2,
			runafter:  0,
			want:      "0",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			for i := 0; i < tc.runbefore; i++ {
				exampleFunc()
				time.Sleep(10 * time.Millisecond)
			}

			err := runtime.Enable(tc.fp, tc.desc)
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Disable(tc.fp)
			for i := 0; i < tc.runafter; i++ {
				exampleFunc()
				time.Sleep(10 * time.Millisecond)
			}
			count, err := runtime.StatusCount(tc.fp)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Compare(tc.want, count) != 0 {
				t.Fatal("counter is not properly incremented")
			}
		})
	}
}

func TestResetingCounterOnTerm(t *testing.T) {
	testcases := []struct {
		name      string
		fp        string
		desc      string
		newdesc   string
		runbefore int
		runafter  int
		want      string
	}{
		{
			name:      "Change and Reset Counter",
			fp:        "runtime_test/ExampleString",
			desc:      `10*sleep(10)->1*return("abc")`,
			newdesc:   "sleep(10)",
			runbefore: 2,
			runafter:  3,
			want:      "3",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := runtime.Enable(tc.fp, tc.desc)
			if err != nil {
				t.Fatal(err)
			}

			for i := 0; i < tc.runbefore; i++ {
				exampleFunc()
				time.Sleep(10 * time.Millisecond)
			}
			err = runtime.Enable(tc.fp, tc.newdesc)
			if err != nil {
				t.Fatal(err)
			}
			defer runtime.Disable(tc.fp)

			for i := 0; i < tc.runafter; i++ {
				exampleFunc()
				time.Sleep(10 * time.Millisecond)
			}
			count, err := runtime.StatusCount(tc.fp)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Compare(tc.want, count) != 0 {
				t.Fatal("counter is not properly incremented")
			}
		})
	}

}

func exampleFunc() string {
	if vExampleString, __fpErr := __fp_ExampleString.Acquire(); __fpErr == nil { //nolint:stylecheck
		defer __fp_ExampleString.Release()                   //nolint:stylecheck
		ExampleString, __fpTypeOK := vExampleString.(string) //nolint:stylecheck
		if !__fpTypeOK {                                     //nolint:stylecheck
			goto __badTypeExampleString //nolint:stylecheck
		}
		return ExampleString
	__badTypeExampleString: //nolint:stylecheck
		__fp_ExampleString.BadType(vExampleString, "string") //nolint:stylecheck
	}
	return "example"
}
