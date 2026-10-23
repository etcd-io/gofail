package main

import (
	"encoding/json"
	"fmt"
	"go.etcd.io/gofail/integration/server/failpoints"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
)

var funcMap = map[string]interface{}{
	"ExampleFunc":        failpoints.ExampleFunc,
	"ExampleOneLineFunc": failpoints.ExampleOneLineFunc,
	"ExampleLabelsFunc":  failpoints.ExampleLabelsFunc,
}

func callFuncByName(name string, args []string) (interface{}, error) {
	fn, exists := funcMap[name]
	if !exists {
		return nil, fmt.Errorf("function %s does not exist", name)
	}

	fnValue := reflect.ValueOf(fn)
	expectedNArgs := fnValue.Type().NumIn()
	gotNArgs := len(args)
	if expectedNArgs != gotNArgs {
		return nil, fmt.Errorf("wrong number of arguments for function %s. "+
			"Expected %d, got %d", name, expectedNArgs, gotNArgs)
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}

	result := fnValue.Call(in)
	if len(result) == 0 {
		return nil, nil
	}
	return result[0].Interface(), nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request: %v", r)
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.Error(w, "invalid URL path", http.StatusBadRequest)
		return
	}
	funcName := pathParts[2]
	args := r.URL.Query()["args"]

	result, err := callFuncByName(funcName, args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(response)
	if err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Port number is required as a command line argument")
	}
	port := os.Args[1]

	http.HandleFunc("/call/", handler)
	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
