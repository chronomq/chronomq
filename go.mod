module github.com/chronomq/chronomq

go 1.13

require (
	code.cloudfoundry.org/bytefmt v0.0.0-20200125003136-cc367df7c24e
	github.com/DataDog/datadog-go v2.2.0+incompatible
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/onsi/ginkgo v1.7.0
	github.com/onsi/gomega v1.4.3
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/satori/go.uuid v1.2.0
	github.com/seiflotfy/cuckoofilter v0.0.0-20190302225222-764cb5258d9b
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/syndtr/goleveldb v0.0.0-20181128100959-b001fa50d6b2
	gocloud.dev v0.18.0
	gopkg.in/yaml.v2 v2.2.2 // indirect
)

replace gopkg.in/fsnotify.v1 v1.4.7 => gopkg.in/fsnotify/fsnotify.v1 v1.4.7
