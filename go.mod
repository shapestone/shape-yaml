module github.com/shapestone/shape-yaml

go 1.25

require (
	github.com/shapestone/shape-core v0.9.2
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/google/uuid v1.6.0 // indirect

replace github.com/shapestone/shape-core => ../shape-core
