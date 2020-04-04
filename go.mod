module github.com/chronomq/chronomq

go 1.14

require (
	cloud.google.com/go v0.39.0
	code.cloudfoundry.org/bytefmt v0.0.0-20200125003136-cc367df7c24e
	github.com/Azure/azure-pipeline-go v0.2.1
	github.com/Azure/azure-storage-blob-go v0.8.0
	github.com/DataDog/datadog-go v2.2.0+incompatible
	github.com/aws/aws-sdk-go v1.19.45
	github.com/ckaznocha/protoc-gen-lint v0.2.1
	github.com/dgryski/go-metro v0.0.0-20180109044635-280f6062b5bc
	github.com/ghodss/yaml v1.0.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.3.5
	github.com/google/uuid v1.1.1
	github.com/google/wire v0.3.0
	github.com/googleapis/gax-go v2.0.2+incompatible
	github.com/googleapis/gax-go/v2 v2.0.4
	github.com/grpc-ecosystem/grpc-gateway v1.14.3
	github.com/hpcloud/tail v1.0.0
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/mattn/go-ieproxy v0.0.0-20190610004146-91bb50d98149
	github.com/onsi/ginkgo v1.7.0
	github.com/onsi/gomega v1.4.3
	github.com/pkg/errors v0.8.1
	github.com/rs/zerolog v1.15.0
	github.com/satori/go.uuid v1.2.0
	github.com/seiflotfy/cuckoofilter v0.0.0-20190302225222-764cb5258d9b
	github.com/spf13/cobra v0.0.3
	github.com/spf13/pflag v1.0.5
	github.com/syndtr/goleveldb v0.0.0-20181128100959-b001fa50d6b2
	go.opencensus.io v0.22.0
	gocloud.dev v0.18.0
	golang.org/x/net v0.0.0-20191002035440-2ec189313ef0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/sys v0.0.0-20190620070143-6f217b454f45
	golang.org/x/text v0.3.2
	golang.org/x/xerrors v0.0.0-20190717185122-a985d3407aa7
	google.golang.org/api v0.6.0
	google.golang.org/appengine v1.6.1
	google.golang.org/genproto v0.0.0-20200319113533-08878b785e9c
	google.golang.org/grpc v1.28.0
	gopkg.in/fsnotify.v1 v1.4.7
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7
	gopkg.in/yaml.v2 v2.2.3
)

replace gopkg.in/fsnotify.v1 v1.4.7 => gopkg.in/fsnotify/fsnotify.v1 v1.4.7
