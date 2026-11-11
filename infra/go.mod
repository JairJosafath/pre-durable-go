// This file exists only to keep the Go tool out of infra/: node_modules
// contains Go templates shipped with the CDK CLI, which `go test ./...`
// would otherwise try to build.
module gdf/infra

go 1.27
