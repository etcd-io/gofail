package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

// portAssignments is a struct that holds the ports for the gofail and server servers
type portAssignments struct {
	// gofailPort is the port the gofail failpoint control API is running on
	gofailPort int

	// serverPort is the port the server test status API is running on
	serverPort int
}

func getOpenPorts(count int) ([]int, error) {
	ports := make([]int, 0, count)
	for i := 0; i < count; i++ {
		listener, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			return nil, err
		}

		// We want to keep listeners open until loop finishes so we don't accidentally reuse a port.
		defer listener.Close()

		addr := listener.Addr().(*net.TCPAddr)
		ports = append(ports, addr.Port)
	}
	return ports, nil
}

type testRequest interface {
	// AssertResponse is the heart of the test logic. It sends the request to the server,
	// then checks the response matches expectations.
	AssertResponse(t *testing.T)
	// SetupPortAssignments is a helper for setting correct request.port value
	SetupPortAssignments(ports portAssignments)
}

type response struct {
	statusCode int
	body       string
}

type request struct {
	port       int
	methodType string
	endpoint   string
	args       []string
	expected   response
}

// serverTestRequest is a test request for the server test status API
type serverTestRequest struct {
	request
}

func (s *serverTestRequest) AssertResponse(t *testing.T) {
	t.Helper()

	// See main.go for endpoint definitions
	endpoint := "call/" + s.endpoint
	if len(s.args) > 0 {
		// http server interprets "?args=" as having a single argument of value ""
		endpoint += "?args=" + strings.Join(s.args, ",")
	}

	methodType := http.MethodGet
	if s.methodType != "" {
		methodType = s.methodType
	}
	body, statusCode, err := sendRequest(t, s.port, methodType, endpoint, []byte{})
	assert.NoError(t, err)
	assert.Equal(t, s.expected.statusCode, statusCode)
	assert.Equal(t, s.expected.body, body)
}

func (s *serverTestRequest) SetupPortAssignments(ports portAssignments) {
	s.port = ports.serverPort
}

// gofailTestRequest is a test request for the gofail server control API
type gofailTestRequest struct {
	request

	// requestType is the kind of request we're making to the failpoint control API.
	// These correspond to the operations described in the "HTTP endpoint" section of the README.
	//
	// Valid values are put, failpoints, listall, list, count, deactivate
	// * put: sets the value of a failpoint, enabling it
	// * failpoints: sets the values of multiple failpoints at once
	// * listall: lists all failpoints and their values
	// * list: lists the value of a single failpoint
	// * count: gets the number of times a failpoint has been hit
	// * deactivate: clears a given failpoint
	requestType string
}

func (g *gofailTestRequest) AssertResponse(t *testing.T) {
	t.Helper()
	require.NotEqual(t, g.port, 0, "port is not set")

	endpoint := g.endpoint
	payload := ""
	var methodType string

	switch g.requestType {
	case "put":
		methodType = http.MethodPut
		assert.NotEmpty(t, g.endpoint, "endpoint should not be empty for `put' request type")
		assert.Equalf(t, len(g.args), 1, "args should have exactly one element for `put' request type")
		payload = g.args[0]
		break
	case "failpoints":
		methodType = http.MethodPut
		assert.Equalf(t, g.endpoint, "", "endpoint should be empty for `failpoints' request type")
		endpoint = "failpoints"
		payload = strings.Join(g.args, ";")
		break
	case "listall":
		methodType = http.MethodGet
		assert.Equalf(t, g.endpoint, "", "endpoint should be empty for `listall' request type")
		assert.Nil(t, g.args, "args should be nil for `listall' request type")
		endpoint = ""
		break
	case "list":
		methodType = http.MethodGet
		assert.NotEmpty(t, g.endpoint, "endpoint should not be empty for `list' request type")
		assert.Nil(t, g.args, "args should be nil for `list' request type")
		break
	case "count":
		methodType = http.MethodGet
		assert.NotEmpty(t, g.endpoint, "endpoint should not be empty for `count' request type")
		assert.Nil(t, g.args, "args should be nil for `count' request type")
		endpoint += "/count"
	case "deactivate":
		methodType = http.MethodDelete
		assert.NotEmpty(t, g.endpoint, "endpoint should not be empty for `delete' request type")
		assert.Nil(t, g.args, "args should be nil for `deactivate' request type")
	default:
		t.Errorf("unknown request type: %s", g.requestType)
		return
	}

	body, statusCode, err := sendRequest(t, g.port, methodType, endpoint, []byte(payload))
	assert.NoError(t, err)
	assert.Equal(t, g.expected.statusCode, statusCode)

	if g.requestType != "listall" {
		assert.Equal(t, g.expected.body, body)
	} else {
		// listall responses don't guarantee an order, so we need a more sophisticated test.

		// This is a pretty naive way to convert strings to maps, and will break
		// if any of the keys/values contain the separator characters.
		stringToMap := func(s, entrySep, kvSep string) map[string]string {
			result := make(map[string]string)
			entries := strings.Split(s, entrySep)
			for _, entry := range entries {
				kv := strings.SplitN(entry, kvSep, 2)
				if len(kv) == 2 {
					result[kv[0]] = kv[1]
				}
			}
			return result
		}

		expected := stringToMap(g.expected.body, "\n", "=")
		actual := stringToMap(body, "\n", "=")
		if !reflect.DeepEqual(expected, actual) {
			t.Errorf(
				"listall response did not match:\n\tExpected:\t%#v\n\tActual:\t\t%#v",
				expected, actual)
		}
	}
}

func (g *gofailTestRequest) SetupPortAssignments(ports portAssignments) {
	g.port = ports.gofailPort
}

// sendRequest is the core helper method
func sendRequest(t *testing.T, port int, method string, endpoint string, data []byte) (string, int, error) {
	t.Helper()

	url := fmt.Sprintf("http://localhost:%d/%s", port, endpoint)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return "", 0, err
	}

	t.Logf("Sending request: %s %s %s", method, url, data)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer assert.NoError(t, req.Body.Close())

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	return string(body), resp.StatusCode, nil
}

// request generation helpers

func rgListAllSuccess(expected string) testRequest {
	return &gofailTestRequest{
		requestType: "listall",
		request: request{
			expected: response{
				statusCode: 200,
				body:       expected,
			},
		},
	}
}

func rgCountSuccess(endpoint string, expected int) testRequest {
	return &gofailTestRequest{
		requestType: "count",
		request: request{
			endpoint: endpoint,
			expected: response{
				statusCode: 200,
				body:       fmt.Sprintf("%d", expected),
			},
		},
	}
}

func rgTestServerSuccess(endpoint string, expected string, args ...string) testRequest {
	return &serverTestRequest{
		request: request{
			endpoint: endpoint,
			args:     args,
			expected: response{
				statusCode: 200,
				body:       "\"" + expected + "\"",
			},
		},
	}
}

// TestAll is a comprehensive test suite for the server and gofail control APIs.
// Each test consists of a series of requests and expected responses to send to either API.
// The tests share a single server process which preserves state between tests,
// so they must be run in order.  We could potentially change this to spawn separate
// server instances per test, which would isolate the test cases, at the cost of making
// some individual cases more complex.
//
// Some notes for maintainers:
//   - When debugging in an IDE, make sure to run `make gofail-enable` in the root directory
//     to enable the gofail failpoint control API.
//   - Make sure to run all tests every time, as later tests depend on earlier ones.
//     Fortunately, they run very quickly.  If necessary, you can reset state by sending
//     deactivate requests for the relevant failpoints.
//   - You can spawn the main.go server manually and attach a debugger to it, then hack
//     getOpenPorts() to return the relevant ports.  This will allow you to effectively
//     debug client-server interactions. You may need to increase the timeout variable
//     below.
//     That said, stdout/err messages from the server process are emitted in the test
//     output, and are usually sufficient for debugging.
func TestAll(t *testing.T) {
	timeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ports, err := getOpenPorts(2)
	require.NoError(t, err)
	pas := portAssignments{gofailPort: ports[0], serverPort: ports[1]}

	waitForServer := make(chan any)

	// Spawn server.go in a goroutine
	go func() {
		defer func() {
			println("Server goroutine exited")
		}()

		pipeToStdout := func(reader io.ReadCloser) {
			scanner := bufio.NewScanner(reader)
			for scanner.Scan() {
				fmt.Println("[Test Server] " + scanner.Text())
			}
		}

		cmd := exec.CommandContext(ctx, "go", "run", "main.go", fmt.Sprintf("%d", pas.serverPort))

		stdoutReader, err := cmd.StdoutPipe()
		require.NoError(t, err)
		stderrReader, err := cmd.StderrPipe()
		require.NoError(t, err)
		go pipeToStdout(stdoutReader)
		go pipeToStdout(stderrReader)

		cmd.WaitDelay = timeout
		cmd.Env = append(os.Environ(), fmt.Sprintf("GOFAIL_HTTP=:%d", pas.gofailPort))
		t.Logf("Starting server: GOFAIL_HTTP=:%d %v", pas.gofailPort, cmd)

		err = cmd.Start()
		require.NoError(t, err)
		t.Logf("Waiting for server to be available, pid %d", cmd.Process.Pid)
		assert.Eventuallyf(t, func() bool {
			_, statusCode, _ := sendRequest(t, pas.serverPort, http.MethodGet, "call/ExampleFunc", []byte{})
			return statusCode == 200
		}, timeout, 100*time.Millisecond, "Server did not become available in time")

		// signal server start
		waitForServer <- struct{}{}

		t.Logf("Waiting for server to exit, pid %d", cmd.Process.Pid)
		err = cmd.Wait()

		// We always stop the server by cancelling the context, so this should always be sigkill
		t.Logf("Server exited (sigkill is expected) with error: %v, context cancellation reason %v", err, ctx.Err())
		assert.Error(t, err)

		// signal server exit
		waitForServer <- struct{}{}
	}()

	defer func() {
		// Give it some time to drain IO pipes
		time.Sleep(1 * time.Second)
		cancel()

		// Wait for the server to exit
		<-waitForServer
	}()

	// Wait for server to start and be ready
	<-waitForServer

	tests := []struct {
		name     string
		requests []testRequest
	}{
		{
			name: "Empty listall",
			requests: []testRequest{
				rgListAllSuccess("ExampleLabels=\nExampleOneLine=\nExampleString=\n"),
			},
		},
		{
			name: "list for disabled failpoint returns 404",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "list",
					request: request{
						endpoint: "ExampleString",
						expected: response{
							statusCode: 404,
							body:       "failed to GET: failpoint: failpoint is disabled\n\n",
						},
					},
				},
			},
		},
		{
			name: "list for invalid failpoint returns 404",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "list",
					request: request{
						endpoint: "InvalidFailpoint",
						expected: response{
							statusCode: 404,
							body:       "failed to GET: failpoint: failpoint does not exist\n\n",
						},
					},
				},
			},
		},
		{
			name: "count fails for disabled failpoints",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "count",
					request: request{
						endpoint: "ExampleLabels",
						expected: response{
							statusCode: 500,
							body:       "failed to GET: failpoint: failpoint is disabled\n",
						},
					},
				},
			},
		},
		{
			name: "count fails for invalid failpoints",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "count",
					request: request{
						endpoint: "InvalidFailpoint",
						expected: response{
							statusCode: 404,
							body:       "failed to GET: failpoint: failpoint does not exist\n",
						},
					},
				},
			},
		},
		{
			name: "count starts at 0",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "put",
					request: request{
						endpoint: "ExampleString",
						args:     []string{"return(\"fail string\")"},
						expected: response{
							statusCode: 204,
							body:       "",
						},
					},
				},
				rgCountSuccess("ExampleString", 0),
			},
		},
		{
			name: "list works for enabled failpoints",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "list",
					request: request{
						endpoint: "ExampleString",
						expected: response{
							statusCode: 200,
							body:       "return(\"fail string\")\n",
						},
					},
				},
			},
		},
		{
			name: "listall after put",
			requests: []testRequest{
				rgListAllSuccess("ExampleLabels=\nExampleOneLine=\nExampleString=return(\"fail string\")\n"),
			},
		},
		{
			name: "count increments to 1",
			requests: []testRequest{
				rgTestServerSuccess("ExampleFunc", "fail string"),
				rgCountSuccess("ExampleString", 1),
			},
		},
		{
			name: "putting a new value updates an existing failpoint",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "put",
					request: request{
						endpoint: "ExampleString",
						args:     []string{"return(\"new fail string\")"},
						expected: response{
							statusCode: 204,
							body:       "",
						},
					},
				},
				rgCountSuccess("ExampleString", 0),
				rgTestServerSuccess("ExampleFunc", "new fail string"),
				rgCountSuccess("ExampleString", 1),
			},
		},
		{
			name: "deactivate works",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "deactivate",
					request: request{
						endpoint: "ExampleString",
						expected: response{
							statusCode: 204,
							body:       "",
						},
					},
				},
				rgListAllSuccess("ExampleLabels=\nExampleOneLine=\nExampleString=\n"),
				&gofailTestRequest{
					requestType: "list",
					request: request{
						endpoint: "ExampleString",
						expected: response{
							statusCode: 404,
							body:       "failed to GET: failpoint: failpoint is disabled\n\n",
						},
					},
				},
				&gofailTestRequest{
					requestType: "count",
					request: request{
						endpoint: "ExampleString",
						expected: response{
							statusCode: 500,
							body:       "failed to GET: failpoint: failpoint is disabled\n",
						},
					},
				},
				rgTestServerSuccess("ExampleFunc", "example"),
			},
		},
		{
			name: "re-enabling a failpoint resets count",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "put",
					request: request{
						endpoint: "ExampleString",
						args:     []string{"return(\"new fail string\")"},
						expected: response{
							statusCode: 204,
							body:       "",
						},
					},
				},
				&gofailTestRequest{
					requestType: "count",
					request: request{
						endpoint: "ExampleString",
						expected: response{
							statusCode: 200,
							body:       "0",
						},
					},
				},
			},
		},
		{
			name: "failpoints happy path",
			requests: []testRequest{
				rgTestServerSuccess("ExampleFunc", "new fail string"),
				rgTestServerSuccess("ExampleOneLineFunc", "abc"),
				rgTestServerSuccess("ExampleLabelsFunc", "ijjjjjijjjjjijjjjjijjjjjijjjjj"),
				&gofailTestRequest{
					requestType: "failpoints",
					request: request{
						args: []string{
							"ExampleString=1*return(\"fail string1\")->return(\"fail string2\")",
							"ExampleOneLine=return(\"def\")",
							"ExampleLabels=return"},
						expected: response{
							statusCode: 204,
							body:       "",
						},
					},
				},
				rgListAllSuccess(strings.Join([]string{
					"ExampleString=1*return(\"fail string1\")->return(\"fail string2\")",
					"ExampleOneLine=return(\"def\")",
					"ExampleLabels=return"}, "\n") + "\n"),
				rgCountSuccess("ExampleString", 0),
				rgTestServerSuccess("ExampleFunc", "fail string1"),
				rgTestServerSuccess("ExampleFunc", "fail string2"),
				rgTestServerSuccess("ExampleOneLineFunc", "abc"),
				rgTestServerSuccess("ExampleLabelsFunc", "ijijijijij"),
				rgCountSuccess("ExampleString", 2),
				rgCountSuccess("ExampleOneLine", 1),
				rgCountSuccess("ExampleLabels", 5),
			},
		},
		{
			name: "failpoints works with subset of failpoints",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "failpoints",
					request: request{
						args: []string{"ExampleOneLine=return(\"ghi\")"},
						expected: response{
							statusCode: 204,
							body:       "",
						},
					},
				},
				rgListAllSuccess(strings.Join([]string{
					"ExampleString=1*return(\"fail string1\")->return(\"fail string2\")",
					"ExampleOneLine=return(\"ghi\")",
					"ExampleLabels=return"}, "\n") + "\n"),
				rgCountSuccess("ExampleString", 2),
				rgCountSuccess("ExampleOneLine", 0),
				rgCountSuccess("ExampleLabels", 5),
			},
		},
		{
			name: "failpoints ignores invalid failpoints, and processes valid ones",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "failpoints",
					request: request{
						args: []string{
							"ExampleOneLine=return(\"jkl\")",
							"InvalidFailpoint=return",
						},
						expected: response{
							statusCode: 400,
							body:       "fail to set failpoint: failpoint: failpoint does not exist\n",
						},
					},
				},
				rgListAllSuccess(strings.Join([]string{
					"ExampleString=1*return(\"fail string1\")->return(\"fail string2\")",
					"ExampleOneLine=return(\"jkl\")",
					"ExampleLabels=return"}, "\n") + "\n"),
				rgCountSuccess("ExampleString", 2),
				rgCountSuccess("ExampleOneLine", 0),
				rgCountSuccess("ExampleLabels", 5),
			},
		},
		{
			name: "failpoints can reset failpoints",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "failpoints",
					request: request{
						args: []string{
							"ExampleString=off",
							"ExampleOneLine=off",
							"ExampleLabels=off",
						},
						expected: response{
							statusCode: 204,
							body:       "",
						},
					},
				},
				rgListAllSuccess(strings.Join([]string{
					"ExampleString=off",
					"ExampleOneLine=off",
					"ExampleLabels=off",
				}, "\n") + "\n"),
				rgTestServerSuccess("ExampleFunc", "example"),
				rgTestServerSuccess("ExampleOneLineFunc", "abc"),
				rgTestServerSuccess("ExampleLabelsFunc", "ijjjjjijjjjjijjjjjijjjjjijjjjj"),
				rgCountSuccess("ExampleString", 1),
				rgCountSuccess("ExampleOneLine", 1),
				rgCountSuccess("ExampleLabels", 25),
			},
		},
		{
			name: "failpoints can't delete failpoints :(",
			requests: []testRequest{
				&gofailTestRequest{
					requestType: "failpoints",
					request: request{
						args: []string{
							"ExampleString=",
							"ExampleOneLine=",
							"ExampleLabels=",
						},
						expected: response{
							statusCode: 400,
							body:       "fail to set failpoint: failpoint: could not parse terms\n",
						},
					},
				},
				rgListAllSuccess(strings.Join([]string{
					"ExampleString=off",
					"ExampleOneLine=off",
					"ExampleLabels=off",
				}, "\n") + "\n"),
				rgTestServerSuccess("ExampleFunc", "example"),
				rgTestServerSuccess("ExampleOneLineFunc", "abc"),
				rgTestServerSuccess("ExampleLabelsFunc", "ijjjjjijjjjjijjjjjijjjjjijjjjj"),
				rgCountSuccess("ExampleString", 2),
				rgCountSuccess("ExampleOneLine", 2),
				rgCountSuccess("ExampleLabels", 50),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, req := range test.requests {
				req.SetupPortAssignments(pas)
				req.AssertResponse(t)
			}
		})
	}
}
