package main

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
)

func getTlsConfig(useTls, tlsVerify bool, keyPath, certPath, caKeyPath, caCertPath string) (*tls.Config, error) {
	var (
		tlsConfig tls.Config
	)
	if !useTls && !tlsVerify {
		log.Debugf("No TLS")
		return nil, nil
	}
	log.Debugf("Setting up TLS")

	tlsConfig.InsecureSkipVerify = true
	if tlsVerify {
		certPool := x509.NewCertPool()
		file, err := ioutil.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}
		certPool.AppendCertsFromPEM(file)
		tlsConfig.RootCAs = certPool
		tlsConfig.InsecureSkipVerify = false
	}

	_, errCert := os.Stat(certPath)
	_, errKey := os.Stat(keyPath)
	if errCert == nil || errKey == nil {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}
	tlsConfig.MinVersion = tls.VersionTLS10
	return &tlsConfig, nil
}
