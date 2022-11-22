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
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
)

type httpHandler struct{}

func serve(host string) error {
	ln, err := net.Listen("tcp", host)
	if err != nil {
		return err
	}
	go http.Serve(ln, &httpHandler{})
	return nil
}

func (*httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// This prevents all failpoints from being triggered. It ensures
	// the server(runtime) doesn't panic due to any failpoints during
	// processing the HTTP request.
	// It may be inefficient, but correctness is more important than
	// efficiency. Usually users will not enable too many failpoints
	// at a time, so it (the efficiency) isn't a problem.
	failpointsMu.Lock()
	defer failpointsMu.Unlock()

	key := r.RequestURI
	if len(key) == 0 || key[0] != '/' {
		http.Error(w, "malformed request URI", http.StatusBadRequest)
		return
	}
	key = key[1:]

	switch {
	// sets the failpoint
	case r.Method == "PUT":
		v, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed ReadAll in PUT", http.StatusBadRequest)
			return
		}

		fpMap := map[string]string{key: string(v)}
		if strings.EqualFold(key, "failpoints") {
			fpMap, err = parseFailpoints(string(v))
			if err != nil {
				http.Error(w, fmt.Sprintf("fail to parse failpoint: %v", err), http.StatusBadRequest)
				return
			}
		}

		for k, v := range fpMap {
			if err := enable(k, v); err != nil {
				http.Error(w, fmt.Sprintf("fail to set failpoint: %v", err), http.StatusBadRequest)
				return
			}
		}
		writeCodeAndFlush(w, http.StatusNoContent)

	// gets status of the failpoint
	case r.Method == "GET":
		if len(key) == 0 {
			fps := list()
			sort.Strings(fps)
			lines := make([]string, len(fps))
			for i := range lines {
				s, _ := status(fps[i])
				lines[i] = fps[i] + "=" + s
			}
			writeDataAndFlush(w, []byte(strings.Join(lines, "\n")+"\n"))
		} else {
			status, err := status(key)
			if err != nil {
				http.Error(w, "failed to GET: "+err.Error(), http.StatusNotFound)
			}
			writeDataAndFlush(w, []byte(status+"\n"))
		}

	// deactivates a failpoint
	case r.Method == "DELETE":
		if err := disable(key); err != nil {
			http.Error(w, "failed to delete failpoint "+err.Error(), http.StatusBadRequest)
			return
		}
		writeCodeAndFlush(w, http.StatusNoContent)
	default:
		w.Header().Add("Allow", "DELETE")
		w.Header().Add("Allow", "GET")
		w.Header().Set("Allow", "PUT")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeCodeAndFlush(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(http.StatusNoContent)
	flush(w)
}

func writeDataAndFlush(w http.ResponseWriter, data []byte) {
	w.Write(data)
	flush(w)
}

func flush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		// flush before unlocking so a panic failpoint won't
		// take down the http server before it sends the response
		f.Flush()
	}
}
