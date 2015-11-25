FROM phusion/baseimage

RUN sed 's/main$/main universe/' -i /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y git
RUN apt-get install -y wget
RUN apt-get install -y mercurial
RUN apt-get install -y build-essential

ENV PROTOVERSION 2.3.0
ENV PROTODOWNLOAD http://protobuf.googlecode.com/files/protobuf-2.3.0.tar.gz

#install protoc
RUN wget $PROTODOWNLOAD
RUN tar xzvf protobuf-$PROTOVERSION.tar.gz
RUN mv protobuf-$PROTOVERSION protobuf
RUN (cd protobuf && ./configure)
RUN (cd protobuf && make)
# RUN (cd protobuf && make check)
RUN (cd protobuf && make install)
RUN (cd protobuf && ldconfig)
RUN protoc --version || true

ENV GOVERSION 1.3.3
ENV GODOWNLOAD http://golang.org/dl/

#install go from mecurial repository
#RUN hg clone -u go$GOVERSION https://code.google.com/p/go
#ENV GOROOT /go
#RUN (cd $GOROOT/src && ./make.bash)

#download go
ENV GOFILENAME go$GOVERSION.linux-amd64.tar.gz
RUN wget $GODOWNLOAD/$GOFILENAME
RUN tar -C / -xzf $GOFILENAME
RUN rm $GOFILENAME
ENV GOROOT /go

#setup go path
RUN mkdir gopath
ENV GOPATH /gopath
ENV PATH $PATH:$GOPATH/bin:$GOROOT/bin

#setup paths for my repositories
RUN mkdir -p $GOPATH/src/github.com/gogo
RUN mkdir -p $GOPATH/src/github.com/golang
ENV GOGOPROTOPATH $GOPATH/src/github.com/gogo/protobuf
ENV GOGOTESTPATH $GOPATH/src/github.com/gogo/harmonytests
ENV GOPROTOPATH $GOPATH/src/github.com/golang/protobuf

#setup the script to run everytime the docker runs
RUN echo '#!/bin/bash' >> /test.sh
RUN echo 'set -xe' >> /test.sh
RUN echo 'go version' >> /test.sh
RUN echo 'protoc --version || true' >> /test.sh
RUN echo 'git clone https://github.com/gogo/protobuf $GOGOPROTOPATH' >> /test.sh
RUN echo 'cd $GOGOPROTOPATH' >> /test.sh
RUN echo 'make all' >> /test.sh

RUN echo 'git clone https://github.com/golang/protobuf $GOPROTOPATH' >> /test.sh
RUN echo 'cd $GOPROTOPATH' >> /test.sh
RUN echo 'make' >> /test.sh

RUN echo 'git clone https://github.com/gogo/harmonytests $GOGOTESTPATH' >> /test.sh
RUN echo 'cd $GOGOTESTPATH' >> /test.sh
RUN echo 'make regenerate' >> /test.sh
RUN echo 'make test' >> /test.sh
RUN echo 'go version' >> /test.sh
RUN echo 'protoc --version || true' >> /test.sh
RUN chmod +x /test.sh

ENTRYPOINT /test.sh


