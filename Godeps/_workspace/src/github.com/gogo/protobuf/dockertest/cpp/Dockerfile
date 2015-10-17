FROM phusion/baseimage

RUN sed 's/main$/main universe/' -i /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y git
RUN apt-get install -y wget
RUN apt-get install -y mercurial
RUN apt-get install -y build-essential

ENV PROTOVERSION 2.5.0

#install protoc
RUN wget http://protobuf.googlecode.com/files/protobuf-$PROTOVERSION.tar.gz
RUN tar xzvf protobuf-$PROTOVERSION.tar.gz
RUN mv protobuf-$PROTOVERSION protobuf
RUN (cd protobuf && ./configure)
RUN (cd protobuf && make)
# RUN (cd protobuf && make check)
RUN (cd protobuf && make install)
RUN (cd protobuf && ldconfig)
RUN protoc --version || true

ENV GOVERSION 1.3
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
RUN echo 'make' >> /test.sh

RUN echo '#build cpp the same as go' >> /test.sh
RUN echo 'cd $GOPATH/src' >> /test.sh
RUN echo 'protoc -I=.:./github.com/gogo/protobuf/protobuf/ --cpp_out=. ./github.com/gogo/protobuf/gogoproto/gogo.proto' >> /test.sh
RUN echo 'g++ -I$GOPATH/src -c -o $GOPATH/src/github.com/gogo/protobuf/gogoproto/gogo.pb.o $GOPATH/src/github.com/gogo/protobuf/gogoproto/gogo.pb.cc' >> /test.sh
RUN echo 'protoc -I=.:./github.com/gogo/protobuf/protobuf/ --cpp_out=. ./github.com/gogo/protobuf/test/example/example.proto' >> /test.sh
RUN echo 'g++ -I$GOPATH/src -c -o $GOPATH/src/github.com/gogo/protobuf/test/example/example.pb.o $GOPATH/src/github.com/gogo/protobuf/test/example/example.pb.cc' >> /test.sh

RUN echo '#cpp will probably have the google folder with the descriptors in the right place, since it will have protoc cpp code available.' >> /test.sh
RUN echo '#This simulates that by moving our google folder and then running protoc see without the extra path.' >> /test.sh
RUN echo 'mv ./github.com/gogo/protobuf/protobuf/google .' >> /test.sh
RUN echo 'protoc -I=. --cpp_out=. ./github.com/gogo/protobuf/gogoproto/gogo.proto' >> /test.sh
RUN echo 'g++ -I$GOPATH/src -c -o $GOPATH/src/github.com/gogo/protobuf/gogoproto/gogo.pb.o $GOPATH/src/github.com/gogo/protobuf/gogoproto/gogo.pb.cc' >> /test.sh
RUN echo 'protoc -I=. --cpp_out=. ./github.com/gogo/protobuf/test/example/example.proto' >> /test.sh
RUN echo 'g++ -I$GOPATH/src -c -o $GOPATH/src/github.com/gogo/protobuf/test/example/example.pb.o $GOPATH/src/github.com/gogo/protobuf/test/example/example.pb.cc' >> /test.sh

RUN chmod +x /test.sh

ENTRYPOINT /test.sh


