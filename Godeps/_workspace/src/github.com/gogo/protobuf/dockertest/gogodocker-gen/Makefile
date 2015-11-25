all:
	go install .
	gogodocker-gen ./config.json ./Dockerfile ../generated-dockerfiles

clean:
	rm -rf ../generated-dockerfiles
