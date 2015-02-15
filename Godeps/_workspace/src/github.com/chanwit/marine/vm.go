package marine

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Machine struct {
	Name           string
	ID             string
	ForwardingPort string
	IPAddr         string
}

func (m *Machine) CloneN(num int, prefix string) ([]*Machine, error) {
	networkName, err := getNetworkName()
	if err != nil {
		return nil, err
	}

	err = Clone(m.Name, prefix, num, networkName)
	if err != nil {
		return nil, err
	}
	machines := make([]*Machine, num)
	for i := 1; i <= num; i++ {
		name := fmt.Sprintf("%s%03d", prefix, i)
		machines[i-1] = &Machine{Name: name}
	}
	return machines, nil
}

func (m *Machine) Clone(prefix string) (*Machine, error) {
	box, err := m.CloneN(1, prefix)
	return box[0], err
}

func (m *Machine) StartAndWait() error {
	if m.ForwardingPort != "" {
		return StartAndWait(m.Name, m.ForwardingPort)
	}

	out, err := exec.Command(VBOX_MANAGE, "showvminfo", m.Name, "--machinereadable").Output()
	if err != nil {
		return fmt.Errorf("Cannot get vminfo %s", m.Name)
	}
	log.Debugf("Found info : %d length", len(out))
	m.ForwardingPort = ""
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Forwarding") {
			re := regexp.MustCompile(`Forwarding\(\d+\)="ssh,tcp,[0-9\.]*,(\d+),[0-9\.]*,\d+"`)
			result := re.FindStringSubmatch(line)
			if len(result) == 2 {
				m.ForwardingPort = result[1]
				break
			}
		}
	}
	if m.ForwardingPort == "" {
		return fmt.Errorf("Cannot find port: %s", m.Name)
	}
	log.Debugf("Found %s = %s", m.Name, m.ForwardingPort)
	return StartAndWait(m.Name, m.ForwardingPort)
}

func (m *Machine) Run(cmd string, raw ...string) (string, error) {
	if len(raw) == 0 {
		log.Infof("Running: \"%s\"", cmd)
	} else {
		log.Infof(raw[0], raw[1])
	}
	_, sess, err := ConnectToHost("ubuntu@127.0.0.1:"+m.ForwardingPort, "reverse")
	defer sess.Close()
	if err != nil {
		return "", err
	}

	b, err := sess.Output(cmd)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (m *Machine) Sudo(cmd string) (string, error) {
	// "/bin/bash -c 'echo reverse | sudo -S whoami'"
	sudo := fmt.Sprintf("/bin/bash -c 'echo reverse | sudo -S bash -c \"%s\"'", cmd)
	return m.Run(sudo, "Sudo: \"%s\"", cmd)
}

func (m *Machine) SetupIPAddr() error {
	_, err := m.Sudo(`sed -i "\$aauto eth1\niface eth1 inet dhcp\n" /etc/network/interfaces`)
	if err != nil {
		return fmt.Errorf("Cannot sed: %s:", err)
	}

	_, err = m.Sudo("ifup eth1")
	if err != nil {
		return fmt.Errorf("Cannot ifup: %s:", err)
	}

	return nil
}

/*
3: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
    link/ether 08:00:27:c7:d6:16 brd ff:ff:ff:ff:ff:ff
    inet 192.168.99.107/24 brd 192.168.99.255 scope global eth1
       valid_lft forever preferred_lft forever
    inet6 fe80::a00:27ff:fec7:d616/64 scope link
       valid_lft forever preferred_lft forever
*/
func (m *Machine) GetIPAddr() (string, error) {
	if out, err := m.Run("/sbin/ip addr show dev eth1"); err == nil {
		str := string(out)
		lines := strings.Split(str, "\n")
		for _, line := range lines {
			line := strings.TrimSpace(line)
			if strings.HasPrefix(line, "inet") {
				result := strings.Split(line, " ")
				ip := strings.SplitN(result[1], "/", 2)[0]
				m.IPAddr = ip
				return ip, nil
			}
		}
	}

	return "", fmt.Errorf("No IP: ", m.Name)
}

func (m *Machine) Remove() error {
	return Remove(m.Name)
}

func (m *Machine) Stop() error {
	return Stop(m.Name)
}

func (m *Machine) InstallDocker() error {
	_, err := m.Sudo("wget -qO- --no-check-certificate https://get.docker.com/ | bash")
	return err
}

func (m *Machine) InstallGolang() error {
	_, err := m.Sudo("service docker start")
	_, err = m.Sudo("docker pull golang:1.3")
	return err
}

func (m *Machine) BuildSwarm(repo string, commits ...string) error {
	url := fmt.Sprintf("git clone --depth 1 http://github.com/%s", repo)
	if len(commits) == 1 {
		url = fmt.Sprintf("wget -qO- --no-check-certificate https://github.com/%s/archive/%s.tar.gz | tar zxf -", repo, commits[0])
	}
	out, err := m.Sudo(url)
	if err != nil {
		log.Error(out)
		return err
	}
	if len(commits) == 1 {
		_, err = m.Sudo("cd swarm* && sed -i \"s/git.rev-parse...short/echo/\" Dockerfile")
	}
	_, err = m.Sudo("cd swarm* && docker build -t swarm:build .")
	if err != nil {
		return err
	}

	return nil
}

func (m *Machine) Exist() bool {
	return false
}
