module github.com/api7/etcdstore

go 1.16

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.5

require (
	github.com/gorilla/securecookie v1.1.1
	github.com/gorilla/sessions v1.2.1
	github.com/stretchr/testify v1.7.0
	go.etcd.io/etcd/client/v3 v3.5.4
)
