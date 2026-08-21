module github.com/apisentinel/apisentinel/agent

go 1.22.0

require (
	github.com/apisentinel/apisentinel v0.0.0
	github.com/fatih/color v1.17.0
	github.com/spf13/cobra v1.8.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.23.0 // indirect
)

replace github.com/apisentinel/apisentinel => ../backend
