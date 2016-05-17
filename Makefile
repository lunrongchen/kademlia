run:
	export GOPATH=`pwd`
	go install kademlia
	./bin/kademlia localhost:2345 localhost:2345

test:
	go test -v libkademlia
