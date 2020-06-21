module github.com/jw-s/redis-operator

go 1.12

require (
	github.com/go-redis/redis v6.15.2+incompatible
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/json-iterator/go v1.1.6 // indirect
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/urfave/cli v1.20.0
	golang.org/x/crypto v0.0.0-20190426145343-a29dc8fdc734 // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.2 // indirect
	k8s.io/api v0.0.0-20190503110853-61630f889b3c
	k8s.io/apimachinery v0.0.0-20190503221204-7a17edec881a
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190502190224-411b2483e503 // indirect
)

replace k8s.io/apimachinery v0.0.0-20190503221204-7a17edec881a => k8s.io/apimachinery v0.0.0-20190221084156-01f179d85dbc

replace k8s.io/api v0.0.0-20190503110853-61630f889b3c => k8s.io/api v0.0.0-20181128191700-6db15a15d2d3

replace k8s.io/client-go v11.0.0+incompatible => k8s.io/client-go v9.0.0+incompatible
