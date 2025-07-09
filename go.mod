module github.com/clavataai/gosdk

go 1.24.2

toolchain go1.24.3

require (
	github.com/cenkalti/backoff/v4 v4.3.0
	github.com/clavataai/monorail/libs/protobufs v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.10.0
	go.skia.org/infra v0.0.0-20250708213113-87a300faac29
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jcgregorio/logger v0.1.3 // indirect
	github.com/jcgregorio/slog v0.0.0-20190423190439-e6f2d537f900 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/zeebo/bencode v1.0.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Replace directive to use local protobufs from the monorepo
replace github.com/clavataai/monorail/libs/protobufs => ../libs/protobufs
