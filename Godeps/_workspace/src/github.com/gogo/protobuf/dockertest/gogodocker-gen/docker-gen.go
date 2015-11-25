package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const makefile = `
build:
	docker build -t gogodocker_go$GOVERSION_protoc_$PROTOVERSION .

run:
	docker run --rm=true -t -i --name gogocontainer_go$GOVERSION_protoc_$PROTOVERSION gogodocker_go$GOVERSION_protoc_$PROTOVERSION
`

type Versions struct {
	Go    []Go
	Proto []Proto
}

type Go struct {
	Version  string
	Download string
}

type Proto struct {
	Version  string
	Download string
}

func newDocker(content string, goversion string, godownload string, protoversion string, protodownload string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "ENV GOVERSION") {
			lines[i] = "ENV GOVERSION " + goversion
		}
		if strings.HasPrefix(line, "ENV GODOWNLOAD") {
			lines[i] = "ENV GODOWNLOAD " + godownload
		}
		if strings.HasPrefix(line, "ENV PROTOVERSION") {
			lines[i] = "ENV PROTOVERSION " + protoversion
		}
		if strings.HasPrefix(line, "ENV PROTODOWNLOAD") {
			lines[i] = "ENV PROTODOWNLOAD " + protodownload
		}
	}
	return strings.Join(lines, "\n")
}

func newMake(content string, goversion string, protoversion string) string {
	content = strings.Replace(content, "$GOVERSION", goversion, -1)
	return strings.Replace(content, "$PROTOVERSION", protoversion, -1)
}

func mapper(ss1 []string, f func(string) string) []string {
	ss2 := make([]string, len(ss1))
	for i, s := range ss1 {
		ss2[i] = f(s)
	}
	return ss2
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 3 {
		fmt.Fprintf(os.Stderr, "expected three parameters, a config filename followed by a dockerfile filename and an output folder")
		os.Exit(1)
	}
	configFilename := flag.Args()[0]
	dockerFilename := flag.Args()[1]
	outputFolder := flag.Args()[2]
	data, err := ioutil.ReadFile(configFilename)
	if err != nil {
		panic(err)
	}
	dockerdata, err := ioutil.ReadFile(dockerFilename)
	if err != nil {
		panic(err)
	}
	dockerstr := string(dockerdata)
	versions := &Versions{}
	err = json.Unmarshal(data, versions)
	if err != nil {
		panic(err)
	}
	folders := []string{}
	for _, goversion := range versions.Go {
		for _, protoversion := range versions.Proto {
			folderName := "go" + goversion.Version + "_protoc" + protoversion.Version
			path := filepath.Join(outputFolder, folderName)
			err = os.MkdirAll(path, 0755)
			if err != nil {
				panic(err)
			}
			newDockerStr := newDocker(dockerstr, goversion.Version, goversion.Download, protoversion.Version, protoversion.Download)
			filename := filepath.Join(path, "Dockerfile")
			err = ioutil.WriteFile(filename, []byte(newDockerStr), 0644)
			if err != nil {
				panic(err)
			}
			newMakeStr := newMake(makefile, goversion.Version, protoversion.Version)
			filename = filepath.Join(path, "Makefile")
			err = ioutil.WriteFile(filename, []byte(newMakeStr), 0644)
			if err != nil {
				panic(err)
			}
			folders = append(folders, folderName)
		}
	}
	builders := mapper(folders, func(s string) string {
		return fmt.Sprintf("\t(cd %s && make build)", s)
	})
	runners := mapper(folders, func(s string) string {
		return fmt.Sprintf("\t(cd %s && make run)", s)
	})
	makeall := "build:\n" + strings.Join(builders, "\n") + "\nrun:\n" + strings.Join(runners, "\n")
	err = ioutil.WriteFile(filepath.Join(outputFolder, "Makefile"), []byte(makeall), 0644)
	if err != nil {
		panic(err)
	}
}
