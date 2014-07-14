default:
	go test -v . ./backends

deps: godep
	cd swarmd && godep restore

save-deps: godep
	cd swarmd && godep save

godep:
	go get github.com/tools/godep
