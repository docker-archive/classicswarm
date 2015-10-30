// +build linux

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

package exec

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"testing"
	"text/template"

	"k8s.io/kubernetes/pkg/kubelet/network"
)

// The temp dir where test plugins will be stored.
const testPluginPath = "/tmp/fake/plugins/net"

func installPluginUnderTest(t *testing.T, vendorName string, plugName string, execTemplateData *map[string]interface{}) {
	vendoredName := plugName
	if vendorName != "" {
		vendoredName = fmt.Sprintf("%s~%s", vendorName, plugName)
	}
	pluginDir := path.Join(testPluginPath, vendoredName)
	err := os.MkdirAll(pluginDir, 0777)
	if err != nil {
		t.Errorf("Failed to create plugin: %v", err)
	}
	pluginExec := path.Join(pluginDir, plugName)
	f, err := os.Create(pluginExec)
	if err != nil {
		t.Errorf("Failed to install plugin")
	}
	err = f.Chmod(0777)
	if err != nil {
		t.Errorf("Failed to set exec perms on plugin")
	}
	const execScriptTempl = `#!/bin/bash

# If status hook is called print the expected json to stdout
if [ "$1" == "status" ]; then
  echo -n '{
	"ip" : "{{.IPAddress}}"
}'
fi

# Direct the arguments to a file to be tested against later
echo -n $@ &> {{.OutputFile}}
`
	if execTemplateData == nil {
		execTemplateData = &map[string]interface{}{
			"IPAddress":  "10.20.30.40",
			"OutputFile": path.Join(pluginDir, plugName+".out"),
		}
	}

	tObj := template.Must(template.New("test").Parse(execScriptTempl))
	buf := &bytes.Buffer{}
	if err := tObj.Execute(buf, *execTemplateData); err != nil {
		t.Errorf("Error in executing script template - %v", err)
	}
	execScript := buf.String()
	_, err = f.WriteString(execScript)
	if err != nil {
		t.Errorf("Failed to write plugin exec")
	}
	f.Close()
}

func tearDownPlugin(plugName string) {
	err := os.RemoveAll(testPluginPath)
	if err != nil {
		fmt.Printf("Error in cleaning up test: %v", err)
	}
}

func TestSelectPlugin(t *testing.T) {
	// install some random plugin under testPluginPath
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	installPluginUnderTest(t, "", pluginName, nil)

	plug, err := network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), pluginName, network.NewFakeHost(nil))
	if err != nil {
		t.Errorf("Failed to select the desired plugin: %v", err)
	}
	if plug.Name() != pluginName {
		t.Errorf("Wrong plugin selected, chose %s, got %s\n", pluginName, plug.Name())
	}
}

func TestSelectVendoredPlugin(t *testing.T) {
	// install some random plugin under testPluginPath
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	vendor := "mycompany"
	installPluginUnderTest(t, vendor, pluginName, nil)

	vendoredPluginName := fmt.Sprintf("%s/%s", vendor, pluginName)
	plug, err := network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), vendoredPluginName, network.NewFakeHost(nil))
	if err != nil {
		t.Errorf("Failed to select the desired plugin: %v", err)
	}
	if plug.Name() != vendoredPluginName {
		t.Errorf("Wrong plugin selected, chose %s, got %s\n", vendoredPluginName, plug.Name())
	}
}

func TestSelectWrongPlugin(t *testing.T) {
	// install some random plugin under testPluginPath
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	installPluginUnderTest(t, "", pluginName, nil)

	wrongPlugin := "abcd"
	plug, err := network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), wrongPlugin, network.NewFakeHost(nil))
	if plug != nil || err == nil {
		t.Errorf("Expected to see an error. Wrong plugin selected.")
	}
}

func TestPluginValidation(t *testing.T) {
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	installPluginUnderTest(t, "", pluginName, nil)

	// modify the perms of the pluginExecutable
	f, err := os.Open(path.Join(testPluginPath, pluginName, pluginName))
	if err != nil {
		t.Errorf("Nil value expected.")
	}
	err = f.Chmod(0444)
	if err != nil {
		t.Errorf("Failed to set perms on plugin exec")
	}
	f.Close()

	_, err = network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), pluginName, network.NewFakeHost(nil))
	if err == nil {
		// we expected an error here because validation would have failed
		t.Errorf("Expected non-nil value.")
	}
}

func TestPluginSetupHook(t *testing.T) {
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	installPluginUnderTest(t, "", pluginName, nil)

	plug, err := network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), pluginName, network.NewFakeHost(nil))

	err = plug.SetUpPod("podNamespace", "podName", "dockerid2345")
	if err != nil {
		t.Errorf("Expected nil: %v", err)
	}
	// check output of setup hook
	output, err := ioutil.ReadFile(path.Join(testPluginPath, pluginName, pluginName+".out"))
	if err != nil {
		t.Errorf("Expected nil")
	}
	expectedOutput := "setup podNamespace podName dockerid2345"
	if string(output) != expectedOutput {
		t.Errorf("Mismatch in expected output for setup hook. Expected '%s', got '%s'", expectedOutput, string(output))
	}
}

func TestPluginTearDownHook(t *testing.T) {
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	installPluginUnderTest(t, "", pluginName, nil)

	plug, err := network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), pluginName, network.NewFakeHost(nil))

	err = plug.TearDownPod("podNamespace", "podName", "dockerid2345")
	if err != nil {
		t.Errorf("Expected nil")
	}
	// check output of setup hook
	output, err := ioutil.ReadFile(path.Join(testPluginPath, pluginName, pluginName+".out"))
	if err != nil {
		t.Errorf("Expected nil")
	}
	expectedOutput := "teardown podNamespace podName dockerid2345"
	if string(output) != expectedOutput {
		t.Errorf("Mismatch in expected output for teardown hook. Expected '%s', got '%s'", expectedOutput, string(output))
	}
}

func TestPluginStatusHook(t *testing.T) {
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	installPluginUnderTest(t, "", pluginName, nil)

	plug, err := network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), pluginName, network.NewFakeHost(nil))

	ip, err := plug.Status("namespace", "name", "dockerid2345")
	if err != nil {
		t.Errorf("Expected nil got %v", err)
	}
	// check output of status hook
	output, err := ioutil.ReadFile(path.Join(testPluginPath, pluginName, pluginName+".out"))
	if err != nil {
		t.Errorf("Expected nil")
	}
	expectedOutput := "status namespace name dockerid2345"
	if string(output) != expectedOutput {
		t.Errorf("Mismatch in expected output for status hook. Expected '%s', got '%s'", expectedOutput, string(output))
	}
	if ip.IP.String() != "10.20.30.40" {
		t.Errorf("Mismatch in expected output for status hook. Expected '10.20.30.40', got '%s'", ip.IP.String())
	}
}

func TestPluginStatusHookIPv6(t *testing.T) {
	pluginName := fmt.Sprintf("test%d", rand.Intn(1000))
	defer tearDownPlugin(pluginName)
	pluginDir := path.Join(testPluginPath, pluginName)
	execTemplate := &map[string]interface{}{
		"IPAddress":  "fe80::e2cb:4eff:fef9:6710",
		"OutputFile": path.Join(pluginDir, pluginName+".out"),
	}
	installPluginUnderTest(t, "", pluginName, execTemplate)

	plug, err := network.InitNetworkPlugin(ProbeNetworkPlugins(testPluginPath), pluginName, network.NewFakeHost(nil))

	ip, err := plug.Status("namespace", "name", "dockerid2345")
	if err != nil {
		t.Errorf("Expected nil got %v", err)
	}
	// check output of status hook
	output, err := ioutil.ReadFile(path.Join(testPluginPath, pluginName, pluginName+".out"))
	if err != nil {
		t.Errorf("Expected nil")
	}
	expectedOutput := "status namespace name dockerid2345"
	if string(output) != expectedOutput {
		t.Errorf("Mismatch in expected output for status hook. Expected '%s', got '%s'", expectedOutput, string(output))
	}
	if ip.IP.String() != "fe80::e2cb:4eff:fef9:6710" {
		t.Errorf("Mismatch in expected output for status hook. Expected 'fe80::e2cb:4eff:fef9:6710', got '%s'", ip.IP.String())
	}
}
