module github.com/xtech/definitions-open-mower/golang

go 1.23

require github.com/xtech/xbot_framework/xbot_service_interface_go v0.1.0

require (
	github.com/fxamacker/cbor/v2 v2.9.3 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

// TODO: drop this replace once xbot_service_interface_go has an upstream tag.
replace github.com/xtech/xbot_framework/xbot_service_interface_go => ../../../xbot_framework/xbot_service_interface_go
