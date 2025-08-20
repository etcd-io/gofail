# gofail

[![Build Status](https://travis-ci.com/etcd-io/gofail.svg?branch=master)](https://travis-ci.com/etcd-io/gofail)

An implementation of [failpoints][failpoint] for golang. Please read [design.md](doc/design.md) for a deeper understanding.

[failpoint]: http://www.freebsd.org/cgi/man.cgi?query=fail

## Add a failpoint

Failpoints are special comments that include a failpoint variable declaration and some trigger code,

```go
func someFunc() string {
	// gofail: var SomeFuncString string
	// // this is called when the failpoint is triggered
	// return SomeFuncString
	return "default"
}
```

## Build with failpoints

Building with failpoints will translate gofail comments in place to code that accesses the gofail runtime.

Call gofail in the directory with failpoints to generate gofail runtime bindings, then build as usual,

```sh
gofail enable
go build cmd/
```

The translated code looks something like,

```go
func someFunc() string {
	if vSomeFuncString, __fpErr := __fp_SomeFuncString.Acquire(); __fpErr == nil { SomeFuncString, __fpTypeOK := vSomeFuncString.(string); if !__fpTypeOK { goto __badTypeSomeFuncString} 
		 // this is called when the failpoint is triggered
		 return SomeFuncString; goto __nomockSomeFuncString; __badTypeSomeFuncString: __fp_SomeFuncString.BadType(vSomeFuncString, "string"); __nomockSomeFuncString: };
	return "default"
}
```

To disable failpoints and revert to the original code,

```sh
gofail disable
```

## Triggering a failpoint

After building with failpoints enabled, the program's failpoints can be activated so they may trigger when evaluated.

### Command line

From the command line, trigger the failpoint to set SomeFuncString to `hello`,

```sh
GOFAIL_FAILPOINTS='SomeFuncString=return("hello")' ./cmd
```

Multiple failpoints are set by using ';' for a delimiter,

```sh
GOFAIL_FAILPOINTS='failpoint1=return("hello");failpoint2=sleep(10)' ./cmd
```

### HTTP endpoint

First, enable the HTTP server from the command line:

```sh
GOFAIL_HTTP="127.0.0.1:1234" ./cmd
```

Activate a single failpoint with curl:

```sh
$ curl http://127.0.0.1:1234/SomeFuncString -XPUT -d'return("hello")'
```

Activate multiple failpoints atomically with the special `/failpoints` endpoint. The payload is the same as for `GOFAIL_FAILPOINTS` above:

```sh
$ curl http://127.0.0.1:1234/failpoints -XPUT -d'failpoint1=return("hello");failpoint2=sleep(10)'
```

List all failpoint configurations:

```sh
$ curl http://127.0.0.1:1234/
```

List a single failpoint configuration:

```sh
$ curl http://127.0.0.1:1234/SomeFuncString
```

Retrieve the execution count of a failpoint:

```sh
$curl http://127.0.0.1:1234/SomeFuncString/count -XGET
```

Deactivate a failpoint:

```sh
$ curl http://127.0.0.1:1234/SomeFuncString -XDELETE
```

### Unit tests

From a unit test,

```go
import (
	"testing"

	gofail "go.etcd.io/gofail/runtime"
)

func TestWhatever(t *testing.T) {
	gofail.Enable("SomeFuncString", `return("hello")`)
	defer gofail.Disable("SomeFuncString")
	...
}
```

