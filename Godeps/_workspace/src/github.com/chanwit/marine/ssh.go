package marine

import (
	"golang.org/x/crypto/ssh"
	"strings"
)

func ConnectToHost(host string, pass string) (*ssh.Client, *ssh.Session, error) {
	str := strings.SplitN(host, "@", 2)
	var user string
	if len(str) == 2 {
		user, host = str[0], str[1]
	}

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(pass)},
	}

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}
