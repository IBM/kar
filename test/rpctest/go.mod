module github.com/IBM/kar/test/rpctest

go 1.15

require (
	github.com/IBM/kar/core v1.2.0
	github.com/Shopify/sarama v1.26.4 // indirect
	github.com/cenkalti/backoff/v4 v4.1.1 // indirect
	github.com/gomodule/redigo v1.8.5 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/julienschmidt/httprouter v1.3.0 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7 // indirect
	k8s.io/api v0.21.3 // indirect
	k8s.io/apimachinery v0.21.3 // indirect
)

replace github.com/IBM/kar/core => ../../core/
