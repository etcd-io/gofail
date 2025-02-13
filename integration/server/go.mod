module go.etcd.io/gofail/integration/server

go 1.22

toolchain go1.22.10

require github.com/stretchr/testify v1.10.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.etcd.io/gofail v0.1.1-0.20240328162059-93c579a86c46 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace go.etcd.io/gofail => ./../../
