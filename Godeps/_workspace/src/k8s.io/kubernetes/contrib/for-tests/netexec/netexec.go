/*
Copyright 2014 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	httpPort  = 8080
	udpPort   = 8081
	shellPath = "/bin/sh"
)

type output struct {
	responses []string
	errors    []string
}

func init() {
	flag.IntVar(&httpPort, "http-port", 8080, "HTTP Listen Port")
	flag.IntVar(&udpPort, "udp-port", 8081, "UDP Listen Port")
}

func main() {
	flag.Parse()
	go startUDPServer(udpPort)
	startHTTPServer(httpPort)
}

func startHTTPServer(httpPort int) {
	http.HandleFunc("/shutdown", shutdownHandler)
	http.HandleFunc("/hostName", hostNameHandler)
	http.HandleFunc("/shell", shellHandler)
	http.HandleFunc("/dial", dialHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil))
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	os.Exit(0)
}

func dialHandler(w http.ResponseWriter, r *http.Request) {
	values, err := url.Parse(r.URL.RequestURI())
	if err != nil {
		http.Error(w, fmt.Sprintf("%v", err), http.StatusBadRequest)
		return
	}

	host := values.Query().Get("host")
	port := values.Query().Get("port")
	request := values.Query().Get("request") // hostName
	protocol := values.Query().Get("protocol")
	tryParam := values.Query().Get("tries")
	tries := 1
	if len(tryParam) > 0 {
		tries, err = strconv.Atoi(tryParam)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("tries parameter is invalid. %v", err), http.StatusBadRequest)
		return
	}
	if len(request) == 0 {
		http.Error(w, fmt.Sprintf("request parameter not specified. %v", err), http.StatusBadRequest)
		return
	}
	if len(protocol) == 0 {
		protocol = "http"
	} else {
		protocol = strings.ToLower(protocol)
	}
	if protocol != "http" && protocol != "udp" {
		http.Error(w, fmt.Sprintf("unsupported protocol. %s", protocol), http.StatusBadRequest)
		return
	}

	hostPort := net.JoinHostPort(host, port)
	var udpAddress *net.UDPAddr
	if protocol == "udp" {
		udpAddress, err = net.ResolveUDPAddr("udp", hostPort)
		if err != nil {
			http.Error(w, fmt.Sprintf("host and/or port param are invalid. %v", err), http.StatusBadRequest)
			return
		}
	} else {
		_, err = net.ResolveTCPAddr("tcp", hostPort)
		if err != nil {
			http.Error(w, fmt.Sprintf("host and/or port param are invalid. %v", err), http.StatusBadRequest)
			return
		}
	}
	errors := make([]string, 0)
	responses := make([]string, 0)
	var response string
	for i := 0; i < tries; i++ {
		if protocol == "udp" {
			response, err = dialUDP(request, udpAddress)
		} else {
			response, err = dialHTTP(request, hostPort)
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("%v", err))
		} else {
			responses = append(responses, response)
		}
	}
	output := map[string][]string{}
	if len(response) > 0 {
		output["responses"] = responses
	}
	if len(errors) > 0 {
		output["errors"] = errors
	}
	bytes, err := json.Marshal(output)
	if err == nil {
		fmt.Fprintf(w, string(bytes))
	} else {
		http.Error(w, fmt.Sprintf("response could not be serialized. %v", err), http.StatusExpectationFailed)
	}
}

func dialHTTP(request, hostPort string) (string, error) {
	transport := &http.Transport{}
	httpClient := createHTTPClient(transport)
	resp, err := httpClient.Get(fmt.Sprintf("http://%s/%s", hostPort, request))
	defer transport.CloseIdleConnections()
	if err == nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			return string(body), nil
		}
	}
	return "", err
}

func createHTTPClient(transport *http.Transport) *http.Client {
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
	return client
}

func dialUDP(request string, remoteAddress *net.UDPAddr) (string, error) {
	Conn, err := net.DialUDP("udp", nil, remoteAddress)
	if err != nil {
		return "", fmt.Errorf("udp dial failed. err:%v", err)
	}

	defer Conn.Close()
	buf := []byte(request)
	_, err = Conn.Write(buf)
	if err != nil {
		return "", fmt.Errorf("udp connection write failed. err:%v", err)
	}
	udpResponse := make([]byte, 1024)
	Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	count, err := Conn.Read(udpResponse)
	if err != nil || count == 0 {
		return "", fmt.Errorf("reading from udp connection failed. err:'%v'", err)
	}
	return string(udpResponse[0:count]), nil
}

func shellHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.FormValue("shellCommand"))
	cmdOut, err := exec.Command(shellPath, "-c", r.FormValue("shellCommand")).CombinedOutput()
	output := map[string]string{}
	if len(cmdOut) > 0 {
		output["output"] = string(cmdOut)
	}
	if err != nil {
		output["error"] = fmt.Sprintf("%v", err)
	}
	bytes, err := json.Marshal(output)
	if err == nil {
		fmt.Fprintf(w, string(bytes))
	} else {
		http.Error(w, fmt.Sprintf("response could not be serialized. %v", err), http.StatusExpectationFailed)
	}
}

func hostNameHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, getHostName())
}

// udp server only supports the hostName command.
func startUDPServer(udpPort int) {
	serverAddress, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", udpPort))
	assertNoError(err)
	serverConn, err := net.ListenUDP("udp", serverAddress)
	defer serverConn.Close()
	buf := make([]byte, 1024)

	for {
		n, clientAddress, err := serverConn.ReadFromUDP(buf)
		assertNoError(err)
		receivedText := strings.TrimSpace(string(buf[0:n]))
		if receivedText == "hostName" {
			log.Println("Sending udp hostName response")
			_, err = serverConn.WriteToUDP([]byte(getHostName()), clientAddress)
			assertNoError(err)
		} else if len(receivedText) > 0 {
			log.Println("Unknown udp command received. ", receivedText)
		}
	}
}

func getHostName() string {
	hostName, err := os.Hostname()
	assertNoError(err)
	return hostName
}

func assertNoError(err error) {
	if err != nil {
		log.Fatal("Error occurred. error:", err)
	}
}
