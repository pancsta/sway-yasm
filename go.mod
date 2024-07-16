module github.com/pancsta/sway-yasm

go 1.22.3

// TODO PR
replace github.com/Difrex/gosway/ipc => github.com/pancsta/gosway/ipc v0.0.0-20240714161203-b47bb358f535

require (
	github.com/Difrex/gosway/ipc v0.0.0-20240312143858-20214f4c38d6
	github.com/fsnotify/fsnotify v1.7.0
	github.com/lithammer/dedent v1.1.0
	github.com/pancsta/asyncmachine-go v0.6.1
	github.com/samber/lo v1.39.0
	github.com/spf13/cobra v1.8.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	golang.org/x/exp v0.0.0-20220303212507-bbda1eaf7a17 // indirect
	golang.org/x/sys v0.16.0 // indirect
)
