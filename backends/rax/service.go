//
// Copyright (C) 2014 Rackspace Hosting Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rax

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/beam"
	"github.com/rackspace/gophercloud"
)

const (
	DASS_TARGET_PREFIX = "rdas_target_"
)

var (
	nilOptions = gophercloud.AuthOptions{}

	// ErrNoPassword errors occur when the value of the OS_PASSWORD environment variable cannot be determined.
	ErrNoPassword = fmt.Errorf("Environment variable OS_PASSWORD or OS_API_KEY needs to be set.")
)

// On status callback for when waiting on cloud actions
type onStatus func(details *gophercloud.Server) error

// copied from backends/forward.go
func copyOutput(sender beam.Sender, reader io.Reader, tag string) {
	chunk := make([]byte, 4096)
	for {
		n, err := reader.Read(chunk)
		if n > 0 {
			sender.Send(&beam.Message{Verb: beam.Log, Args: []string{tag, string(chunk[0:n])}})
		}
		if err != nil {
			message := fmt.Sprintf("Error reading from stream: %v", err)
			sender.Send(&beam.Message{Verb: beam.Error, Args: []string{message}})
			break
		}
	}
}



// Shamelessly taken from docker core
func RandomString() string {
	id := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, id)
	if err != nil {
		panic(err) // This shouldn't happen
	}
	return hex.EncodeToString(id)
}

// Gets the user's auth options from the openstack env variables
func getAuthOptions(ctx *cli.Context) (string, gophercloud.AuthOptions, error) {
	provider := ctx.String("auth-url")
	username := ctx.String("auth-user")
	apiKey := os.Getenv("OS_API_KEY")
	password := os.Getenv("OS_PASSWORD")
	tenantId := os.Getenv("OS_TENANT_ID")
	tenantName := os.Getenv("OS_TENANT_NAME")

	if provider == "" {
		return "", nilOptions, fmt.Errorf("Please set an auth URL with the switch '--auth-url'")
	}

	if username == "" {
		return "", nilOptions, fmt.Errorf("Please set an auth user with the switch '--auth-user'")
	}

	if password == "" && apiKey == "" {
		return "", nilOptions, ErrNoPassword
	}

	ao := gophercloud.AuthOptions{
		Username:   username,
		Password:   password,
		ApiKey:     apiKey,
		TenantId:   tenantId,
		TenantName: tenantName,
	}

	return provider, ao, nil
}
