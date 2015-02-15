package marine

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

var VBOX_MANAGE = "VBoxManage"

func Exist(name string) (bool, error) {
	name = "\"" + name + "\""
	cmd := exec.Command(VBOX_MANAGE, "list", "vms")
	out, err := cmd.Output()
	if err != nil {
		log.Errorf("Exist: %s\n%s\n%s", err, string(out), cmd)
		return false, err
	}
	str := string(out)
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, name) {
			return true, nil
		}
	}
	return false, nil
}

func Export(name string, outfile string) error {
	cmd := exec.Command(VBOX_MANAGE, "export", name,
		"--output", outfile,
		"--ovf20")
	_, err := cmd.Output()
	if err == nil {
		log.Infof("Exported \"%s\" to %s", name, outfile)
	}
	return err
}

func getNetworkName() (string, error) {
	hostOnlyNetwork, _ := getOrCreateHostOnlyNetwork(
		net.ParseIP("192.168.123.1"),
		net.IPv4Mask(255, 255, 255, 0),
		net.ParseIP("192.168.123.2"),
		net.ParseIP("192.168.123.100"),
		net.ParseIP("192.168.123.254"))
	return hostOnlyNetwork.Name, nil
}

func Import(file string, memory int, installs ...string) (*Machine, error) {
	if _, err := os.Stat(file); err != nil {
		log.Error("File not found")
		return nil, fmt.Errorf("File %s not found", file)
	}

	basename := path.Base(file)
	name := strings.SplitN(basename, "-", 2)[0]
	cmd := exec.Command(VBOX_MANAGE, "import", file,
		"--vsys", "0", "--vmname", name,
		"--memory", fmt.Sprintf("%d", memory),
	)
	out, err := cmd.Output()
	if err == nil {
		log.Infof("Imported \"%s\"", name)
	}
	if err != nil {
		log.Errorf("Import: %s\n%s\n%s", err, string(out), cmd)
		return nil, err
	}

	networkName, err := getNetworkName()
	if err != nil {
		log.Errorf("Import: network not found: %s", err)
	}
	Modify(name, networkName, 0)

	m := &Machine{Name: name, ForwardingPort: "52200"}
	if len(installs) > 0 {
		m.StartAndWait()
		for _, i := range installs {
			switch i {
			case "docker":
				m.InstallDocker()
			case "golang":
				m.InstallGolang()
			}
		}
		m.Stop()
	}
	_, err = exec.Command(VBOX_MANAGE, "snapshot", name, "take", "origin").Output()
	log.Infof("Snapshot \"%s/origin\" taken", name)
	return m, err
}

func Modify(name string, adapter string, i int) error {
	err := exec.Command(VBOX_MANAGE, "modifyvm", name,
		"--natpf1", "delete", "ssh").Run()

	out, err := exec.Command(VBOX_MANAGE, "modifyvm", name,
		"--natpf1", fmt.Sprintf("ssh,tcp,127.0.0.1,%d,,22", 52200+i),
		"--nic2", "hostonly",
		"--hostonlyadapter2", adapter,
		"--cableconnected2", "on",
		"--nicpromisc2", "allow-vms",
	).Output()
	if err == nil {
		log.Infof("Modified nic2 for \"%s\"", name)
	} else {
		log.Errorf("Error modify nic2 for \"%s\" to %s \n%s", name, adapter, string(out))
	}
	return err
}

func Clone(baseName string, prefix string, num int, adapter string) error {
	for i := 1; i <= num; i++ {
		name := fmt.Sprintf("%s%03d", prefix, i)
		cmd := exec.Command(VBOX_MANAGE, "clonevm",
			baseName,
			"--snapshot", "origin",
			"--options", "link",
			"--name", name,
			"--register")
		out, err := cmd.Output()
		if err != nil {
			return err
		} else {
			err = Modify(name, adapter, i)
			log.Infof("Clone: %s", strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func Remove(args ...string) error {
	for _, name := range args {
		if name == "base" {
			err := exec.Command(VBOX_MANAGE, "snapshot", "base", "delete", "origin").Run()
			if err != nil {
				log.Info("Removed snapshot \"base/origin\"")
			}
		}
		cmd := exec.Command(VBOX_MANAGE, "unregistervm", name, "--delete")
		_, err := cmd.Output()
		log.Infof("Removed \"%s\"", name)
		if err != nil {
			return err
		}
	}
	return nil
}

func StartAndWait(name string, port string) error {
	err := exec.Command(VBOX_MANAGE, "startvm", name, "--type", "headless").Run()
	log.Infof("Started \"%s\"", name)
	if err != nil {
		return err
	}
	err = WaitForTCP("127.0.0.1:" + port)
	if err == nil {
		log.Infof("VM \"%s\" ready to connect", name)
	}
	return err
}

func Stop(name string) error {
	err := exec.Command(VBOX_MANAGE, "controlvm", name, "acpipowerbutton").Run()
	if err == nil {
		log.Infof("Stopping \"%s\"", name)
	}

	for {
		st := GetState(name)
		if st == "poweroff" {
			log.Infof("VM \"%s\" is now %s", name, st)
			break
		} else if st == "error" {
			return fmt.Errorf("GetState: %s", st)
		}
		time.Sleep(1 * time.Second)
	}

	return err
}

func GetState(name string) string {
	out, err := exec.Command(VBOX_MANAGE, "showvminfo", name, "--machinereadable").Output()
	if err != nil {
		return "exec error"
	}
	str := string(out)
	lines := strings.Split(str, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "VMState=") {
			v := strings.Split(line, "=")[1]
			return v[1 : len(v)-1]
		}
	}
	return "unknown error"
}

func WaitForTCP(addr string) error {
	for {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			continue
		}
		defer conn.Close()
		if _, err = conn.Read(make([]byte, 1)); err != nil {
			continue
		}
		break
	}
	return nil
}
