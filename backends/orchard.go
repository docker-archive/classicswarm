package backends

import (
	"github.com/docker/libswarm"
	"github.com/orchardup/go-orchard/api"

	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
)

func Orchard() libswarm.Sender {
	backend := libswarm.NewServer()
	backend.OnSpawn(func(cmd ...string) (libswarm.Sender, error) {
		if len(cmd) != 2 {
			return nil, fmt.Errorf("orchard: spawn expects 2 arguments: API token and name of host")
		}
		apiToken, hostName := cmd[0], cmd[1]

		apiClient := &api.HTTPClient{
			BaseURL: "https://api.orchardup.com/v2",
			Token:   apiToken,
		}

		host, err := apiClient.GetHost(hostName)
		if err != nil {
			return nil, err
		}

		url := fmt.Sprintf("tcp://%s:4243", host.IPAddress)
		tlsConfig, err := getTLSConfig([]byte(host.ClientCert), []byte(host.ClientKey))
		if err != nil {
			return nil, err
		}

		backend := DockerClientWithConfig(&DockerClientConfig{
			Scheme:          "https",
			URLHost:         host.IPAddress,
			TLSClientConfig: tlsConfig,
		})
		forwardBackend := libswarm.AsClient(backend)
		forwardInstance, err := forwardBackend.Spawn(url)
		if err != nil {
			return nil, err
		}

		return forwardInstance, nil
	})
	return backend
}

func getTLSConfig(clientCertPEMData, clientKeyPEMData []byte) (*tls.Config, error) {
	certPool := x509.NewCertPool()

	certChainPath := os.Getenv("ORCHARD_HOST_CA")
	if certChainPath != "" {
		certChainData, err := ioutil.ReadFile(certChainPath)
		if err != nil {
			return nil, err
		}
		certPool.AppendCertsFromPEM(certChainData)
	} else {
		certPool.AppendCertsFromPEM([]byte(orchardCerts))
	}

	clientCert, err := tls.X509KeyPair(clientCertPEMData, clientKeyPEMData)
	if err != nil {
		return nil, err
	}

	config := new(tls.Config)
	config.RootCAs = certPool
	config.Certificates = []tls.Certificate{clientCert}
	config.BuildNameToCertificate()

	return config, nil
}

var orchardCerts string = `-----BEGIN CERTIFICATE-----
MIIDizCCAnOgAwIBAgIJANOkcdAljaXsMA0GCSqGSIb3DQEBBQUAMFwxCzAJBgNV
BAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xIjAgBgNVBAoMGU9yY2hhcmQgTGFib3Jh
dG9yaWVzIEx0ZC4xGDAWBgNVBAMMD09yY2hhcmQgUm9vdCBDQTAeFw0xMzA5MTAx
OTU4MDZaFw0xODA5MDkxOTU4MDZaMFwxCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZM
b25kb24xIjAgBgNVBAoMGU9yY2hhcmQgTGFib3JhdG9yaWVzIEx0ZC4xGDAWBgNV
BAMMD09yY2hhcmQgUm9vdCBDQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAKE2uHYhJyUxTa/DgXE2Ru85RJptv7YypqL6/hEthhSb30e8c/UZ2RGAupgr
2KhP2B/c78d7hMIX09rbc/Z7TpnV4T3ERuguDJ6jz6NjLKDsW8VVBMx4imcE0hHB
ZrhO+cEuBBufw+iW83uNIkzzIVVlgZ6o9jGXEY61D+SNNxtsMEIkjh+5/JxdeRvK
PSHhBJ2VJpCRWpvuhEuCc5Qlz6PkWEbCchEe8Mzy6Zy8FzT4q4t+ztryWTUavR9s
8lv3N7vHMo4R/r1M+VtjlXzutV8S83avrUQ48woTGBULyfXisbSN6snCXf4VJx8E
UU48FjQBZbKfgV+2ut4f2oUdkNsCAwEAAaNQME4wHQYDVR0OBBYEFBM7bwyZ7n42
9q3EmgaOU7PGa4xtMB8GA1UdIwQYMBaAFBM7bwyZ7n429q3EmgaOU7PGa4xtMAwG
A1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADggEBADZ5YUYc3WfiKWg1VqmBj0EE
wVhy43rId0ruMwcNgyDesCvIDJy5Y9XsIRnRIIeM3tm0MF+fGlmOiN1AvX0KeiTM
7RtPYFawl62aGrDvo0CdZTxCYSRcLvUhgGiftEnqRijawecnk5BhcP+g5Zxe8b+L
DzqbCwG9AQ9M2NAxWdbaBJwAL8qceKklVGWOEpjEYiF4zdNQoLfW+lygVBvKZEwF
By4x8aTPlf8MMMn2ogk4Js8ZcjmvP9fBlA09ecD1DO9lNWMXpDyoB504qTPGoLoj
u6XKHvAXEXvMzLHp7qABqZXDgKIcr8wAqnmqlnFHpLjd+bV7bGMHjGM8EeIdt8I=
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
MIIDjTCCAnWgAwIBAgIJANOkcdAljaXwMA0GCSqGSIb3DQEBBQUAMFwxCzAJBgNV
BAYTAkdCMQ8wDQYDVQQIDAZMb25kb24xIjAgBgNVBAoMGU9yY2hhcmQgTGFib3Jh
dG9yaWVzIEx0ZC4xGDAWBgNVBAMMD09yY2hhcmQgUm9vdCBDQTAeFw0xMzA5MTIx
ODE5MTRaFw0xODA5MTExODE5MTRaMF4xCzAJBgNVBAYTAkdCMQ8wDQYDVQQIDAZM
b25kb24xIjAgBgNVBAoMGU9yY2hhcmQgTGFib3JhdG9yaWVzIEx0ZC4xGjAYBgNV
BAMMEU9yY2hhcmQgRG9ja2VyIENBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIB
CgKCAQEA1T5SrhAZI9N3k8lESi9mYJ9aAqaNESF1qGqWh4Vk2ek0J9X76tPXTK0/
mn9kVMhHFusSHw6EV9imORIdJd9ivdqfMpEPeBuYNuZTNQYsVPP//ZwBPA5+dVOK
dBH+OjgLne8oHIgNM1lRZQlTWrE9FrD11VVnTNcI3VfuDjPD7z5FeYb+gRQx5/u4
gp+xLfglquCzbaPRqQ9FhPB7MFkiQDfZuZieAWZ4QOLJb4za582OX2Gl6mUbIOc7
TQcxeifIwUnSBunq8ER6donjfLy/vJUMBITw4LfzVFpuDnki5FI1DzY+GaVzRSOn
JiGT+WxPH2ydgDieKL1cqB7i+/6o4QIDAQABo1AwTjAdBgNVHQ4EFgQUp/8F8VUv
P+hIDEOf/fzXSujHjYAwHwYDVR0jBBgwFoAUEztvDJnufjb2rcSaBo5Ts8ZrjG0w
DAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQUFAAOCAQEAcbK4gNKRipaDBUwUWU7Z
YXP+npiGEaKxEgBf2J0FFi0yfKRTdACj42vWTZI2m26A1be04xrXZhBKPm4+zN6U
SjEjl+jIKSR7E0QvFv9fFJ0hWl5CDcg8EJdFckgawH2MfSBb4qjN4MDygtOet35Z
VmA7V3AaHa7d2xP+dyER+qP5/ysWzHqliephEpDj4QQIK+bQWwBXj91LhRsyHdn7
VBHS10FcJegbD86SLb85U7zFPZ+vWClDLWwh2hN7ApAGjJgQrfHFhanLi51MIkM2
FHytSi4zdRd+9nEbSVc4t1CL/llaSFk7W77hMFZpq+J+ih6aC0echGQIoXfdHfSM
Qg==
-----END CERTIFICATE-----`
