build: clean
	(cd cpp && make)
	(cd gogodocker-gen && make)
	(cd generated-dockerfiles && make)

clean:
	go clean .
	rm -rf generated-dockerfiles

run: build
	(cd cpp && make run)
	(cd generated-dockerfiles && make run)
