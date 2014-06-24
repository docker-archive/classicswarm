package backends

import (
  "io"
  "os"
  "os/signal"
  "syscall"
  "fmt"
  "net"
  "errors"
  "io/ioutil"
  "github.com/codegangsta/cli"
  "code.google.com/p/go.crypto/ssh"
  "github.com/docker/libswarm/beam"
)

var (
  quit          chan bool
  sshConn       *ssh.Client
  localListener net.Listener
  remoteConn    net.Conn
)

type tunnelConfig struct {
  user string
  host string
  port int
  key  string
  localPort int
  remotePort int
}

type sshClient struct {
  server *beam.Server
  tunnelConfig *tunnelConfig
}

func (c *sshClient) attach(ctx *beam.Message) error {
  ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: c.server})
	<-make(chan struct{})
  return nil
}

func (c *sshClient) loadSshSigner() (ssh.Signer, error) {
  data, err := ioutil.ReadFile(c.tunnelConfig.key)
	if err != nil {
    return nil, err
	}

  key, err := ssh.ParseRawPrivateKey(data)
	if err != nil {
    return nil, err
	}

	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
    return nil, err
	}
  return signer, nil
}

func (c *sshClient) newSshConfig(signer ssh.Signer) (*ssh.ClientConfig) {
  sshConfig := ssh.ClientConfig {
    User: c.tunnelConfig.user,
    Auth: []ssh.AuthMethod {
      ssh.PublicKeys(signer),
    },
  }
  return &sshConfig
}

func (c *sshClient) start(ctx *beam.Message) error {
  signer, err := c.loadSshSigner()
	if err != nil {
		return err
	}

  sshConfig := c.newSshConfig(signer)

  sshConn, err = ssh.Dial("tcp", fmt.Sprintf("%s:%d",
                                             c.tunnelConfig.host,
                                             c.tunnelConfig.port),
                                             sshConfig)
	if err != nil {
		return err
	}

  localListener, err = net.Listen("tcp", fmt.Sprintf("localhost:%d",
                                                  c.tunnelConfig.localPort))
	if err != nil {
		return err
	}

	for {
		incomingConn, err := localListener.Accept()
		if err != nil {
			return err
		}

		go func() {
      remoteConn, err = sshConn.Dial("tcp", fmt.Sprintf("localhost:%d",
                                                        c.tunnelConfig.remotePort))
			if err != nil {
				return
			}

			go copyData(incomingConn, remoteConn)
			go copyData(remoteConn, incomingConn)

		}()

	}
  return nil
}

func copyData(in net.Conn, out net.Conn) {
	buf := make([]byte, 10240)
	for {
		n, err := in.Read(buf)
		if io.EOF == err {
			return
		} else if err != nil {
			return
		}
		out.Write(buf[:n])
	}
}

func handleSignal(client *sshClient) {
  c := make(chan os.Signal, 1)
  signal.Notify(c, os.Interrupt)
  signal.Notify(c, syscall.SIGTERM)
  go func() {
      <-c
			sshConn.Close()
			localListener.Close()
      if (remoteConn != nil) {
			  remoteConn.Close()
      }
      os.Exit(0)
  }()
}

func parseArgs(args []string) (*tunnelConfig, error) {
  app := cli.NewApp()
  app.Flags = []cli.Flag {
    cli.StringFlag{"user",         "",     "REQUIRED - ssh user"},
    cli.StringFlag{"host",         "",     "REQUIRED - ssh host"},
    cli.StringFlag{"key",          "",     "REQUIRED - ssh key - full path"},
    cli.IntFlag   {"port",         22,     "OPTIONAL - ssh port"},
    cli.IntFlag   {"local_port",   4910,   "OPTIONAL - ssh tunnel local port"},
    cli.IntFlag   {"remote_port",  4243,   "OPTIONAL - ssh tunnel remote port"},
  }

  var config *tunnelConfig
  var showHelp = false

  app.Action = func(c *cli.Context) {
    config = &tunnelConfig {
      user:       c.String("user"),
      host:       c.String("host"),
      port:       c.Int("port"),
      key:        c.String("key"),
      localPort:  c.Int("local_port"),
      remotePort: c.Int("remote_port"),
    }
  }

  app.Run(append([]string{"ssh"}, args...))

  if len(args) == 0    || // no args passed
     config == nil     || // help command passed
     config.host == "" ||
     config.key  == "" ||
     config.user == "" {

    showHelp = true
  }

  if showHelp {
    app.Run(append([]string{"ssh"}, "help"))
    return nil, errors.New("help")
  }

  return config, nil
}

func Ssh() beam.Sender {
  backend := beam.NewServer()
  backend.OnSpawn(beam.Handler(func(ctx *beam.Message) error {

    config, err := parseArgs(ctx.Args)

    if err != nil {
      return err
    }

    client := &sshClient{beam.NewServer(), config}

    client.server.OnStart(beam.Handler(client.start))
    client.server.OnAttach(beam.Handler(client.attach))

    handleSignal(client)
    _, err = ctx.Ret.Send(&beam.Message{Verb: beam.Ack, Ret: client.server})

    return err
  }))

  return backend
}
