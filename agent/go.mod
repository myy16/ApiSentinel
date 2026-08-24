module github.com/apisentinel/apisentinel/agent

go 1.25.0

require (
	github.com/apisentinel/apisentinel v0.0.0
	github.com/fatih/color v1.17.0
	github.com/spf13/cobra v1.8.1
	google.golang.org/grpc v1.83.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/apisentinel/apisentinel => ../backend
