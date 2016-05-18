run:
	export GOPATH=`pwd`
	go install kademlia
	./bin/kademlia localhost:2345 localhost:2345

test:
	export GOPATH=`pwd`
	go install kademlia
	go test -v libkademlia
