package kv

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/docker/libkv"
	"github.com/docker/libkv/store"
	libkvmock "github.com/docker/libkv/store/mock"
	"github.com/docker/swarm/discovery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestInitialize(t *testing.T) {
	storeMock, err := libkvmock.New([]string{"127.0.0.1"}, nil)
	assert.NotNil(t, storeMock)
	assert.NoError(t, err)

	d := &Discovery{backend: store.CONSUL}
	d.Initialize("127.0.0.1", 0, 0, nil)
	d.store = storeMock

	s := d.store.(*libkvmock.Mock)
	assert.Len(t, s.Endpoints, 1)
	assert.Equal(t, s.Endpoints[0], "127.0.0.1")
	assert.Equal(t, d.path, defaultDiscoveryPath)

	storeMock, err = libkvmock.New([]string{"127.0.0.1:1234"}, nil)
	assert.NotNil(t, storeMock)
	assert.NoError(t, err)

	d = &Discovery{backend: store.CONSUL}
	d.Initialize("127.0.0.1:1234/path", 0, 0, nil)
	d.store = storeMock

	s = d.store.(*libkvmock.Mock)
	assert.Len(t, s.Endpoints, 1)
	assert.Equal(t, s.Endpoints[0], "127.0.0.1:1234")
	assert.Equal(t, d.path, "path/"+defaultDiscoveryPath)

	storeMock, err = libkvmock.New([]string{"127.0.0.1:1234", "127.0.0.2:1234", "127.0.0.3:1234"}, nil)
	assert.NotNil(t, storeMock)
	assert.NoError(t, err)

	d = &Discovery{backend: store.CONSUL}
	d.Initialize("127.0.0.1:1234,127.0.0.2:1234,127.0.0.3:1234/path", 0, 0, nil)
	d.store = storeMock

	s = d.store.(*libkvmock.Mock)
	if assert.Len(t, s.Endpoints, 3) {
		assert.Equal(t, s.Endpoints[0], "127.0.0.1:1234")
		assert.Equal(t, s.Endpoints[1], "127.0.0.2:1234")
		assert.Equal(t, s.Endpoints[2], "127.0.0.3:1234")
	}
	assert.Equal(t, d.path, "path/"+defaultDiscoveryPath)
}

func TestInitializeWithCerts(t *testing.T) {
	cert := `-----BEGIN CERTIFICATE-----
MIIDCDCCAfKgAwIBAgIICifG7YeiQOEwCwYJKoZIhvcNAQELMBIxEDAOBgNVBAMT
B1Rlc3QgQ0EwHhcNMTUxMDAxMjMwMDAwWhcNMjAwOTI5MjMwMDAwWjASMRAwDgYD
VQQDEwdUZXN0IENBMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1wRC
O+flnLTK5ImjTurNRHwSejuqGbc4CAvpB0hS+z0QlSs4+zE9h80aC4hz+6caRpds
+J908Q+RvAittMHbpc7VjbZP72G6fiXk7yPPl6C10HhRSoSi3nY+B7F2E8cuz14q
V2e+ejhWhSrBb/keyXpcyjoW1BOAAJ2TIclRRkICSCZrpXUyXxAvzXfpFXo1RhSb
UywN11pfiCQzDUN7sPww9UzFHuAHZHoyfTr27XnJYVUerVYrCPq8vqfn//01qz55
Xs0hvzGdlTFXhuabFtQnKFH5SNwo/fcznhB7rePOwHojxOpXTBepUCIJLbtNnWFT
V44t9gh5IqIWtoBReQIDAQABo2YwZDAOBgNVHQ8BAf8EBAMCAAYwEgYDVR0TAQH/
BAgwBgEB/wIBAjAdBgNVHQ4EFgQUZKUI8IIjIww7X/6hvwggQK4bD24wHwYDVR0j
BBgwFoAUZKUI8IIjIww7X/6hvwggQK4bD24wCwYJKoZIhvcNAQELA4IBAQDES2cz
7sCQfDCxCIWH7X8kpi/JWExzUyQEJ0rBzN1m3/x8ySRxtXyGekimBqQwQdFqlwMI
xzAQKkh3ue8tNSzRbwqMSyH14N1KrSxYS9e9szJHfUasoTpQGPmDmGIoRJuq1h6M
ej5x1SCJ7GWCR6xEXKUIE9OftXm9TdFzWa7Ja3OHz/mXteii8VXDuZ5ACq6EE5bY
8sP4gcICfJ5fTrpTlk9FIqEWWQrCGa5wk95PGEj+GJpNogjXQ97wVoo/Y3p1brEn
t5zjN9PAq4H1fuCMdNNA+p1DHNwd+ELTxcMAnb2ajwHvV6lKPXutrTFc4umJToBX
FpTxDmJHEV4bzUzh
-----END CERTIFICATE-----
`
	key := `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEA1wRCO+flnLTK5ImjTurNRHwSejuqGbc4CAvpB0hS+z0QlSs4
+zE9h80aC4hz+6caRpds+J908Q+RvAittMHbpc7VjbZP72G6fiXk7yPPl6C10HhR
SoSi3nY+B7F2E8cuz14qV2e+ejhWhSrBb/keyXpcyjoW1BOAAJ2TIclRRkICSCZr
pXUyXxAvzXfpFXo1RhSbUywN11pfiCQzDUN7sPww9UzFHuAHZHoyfTr27XnJYVUe
rVYrCPq8vqfn//01qz55Xs0hvzGdlTFXhuabFtQnKFH5SNwo/fcznhB7rePOwHoj
xOpXTBepUCIJLbtNnWFTV44t9gh5IqIWtoBReQIDAQABAoIBAHSWipORGp/uKFXj
i/mut776x8ofsAxhnLBARQr93ID+i49W8H7EJGkOfaDjTICYC1dbpGrri61qk8sx
qX7p3v/5NzKwOIfEpirgwVIqSNYe/ncbxnhxkx6tXtUtFKmEx40JskvSpSYAhmmO
1XSx0E/PWaEN/nLgX/f1eWJIlxlQkk3QeqL+FGbCXI48DEtlJ9+MzMu4pAwZTpj5
5qtXo5JJ0jRGfJVPAOznRsYqv864AhMdMIWguzk6EGnbaCWwPcfcn+h9a5LMdony
MDHfBS7bb5tkF3+AfnVY3IBMVx7YlsD9eAyajlgiKu4zLbwTRHjXgShy+4Oussz0
ugNGnkECgYEA/hi+McrZC8C4gg6XqK8+9joD8tnyDZDz88BQB7CZqABUSwvjDqlP
L8hcwo/lzvjBNYGkqaFPUICGWKjeCtd8pPS2DCVXxDQX4aHF1vUur0uYNncJiV3N
XQz4Iemsa6wnKf6M67b5vMXICw7dw0HZCdIHD1hnhdtDz0uVpeevLZ8CgYEA2KCT
Y43lorjrbCgMqtlefkr3GJA9dey+hTzCiWEOOqn9RqGoEGUday0sKhiLofOgmN2B
LEukpKIey8s+Q/cb6lReajDVPDsMweX8i7hz3Wa4Ugp4Xa5BpHqu8qIAE2JUZ7bU
t88aQAYE58pUF+/Lq1QzAQdrjjzQBx6SrBxieecCgYEAvukoPZEC8mmiN1VvbTX+
QFHmlZha3QaDxChB+QUe7bMRojEUL/fVnzkTOLuVFqSfxevaI/km9n0ac5KtAchV
xjp2bTnBb5EUQFqjopYktWA+xO07JRJtMfSEmjZPbbay1kKC7rdTfBm961EIHaRj
xZUf6M+rOE8964oGrdgdLlECgYEA046GQmx6fh7/82FtdZDRQp9tj3SWQUtSiQZc
qhO59Lq8mjUXz+MgBuJXxkiwXRpzlbaFB0Bca1fUoYw8o915SrDYf/Zu2OKGQ/qa
V81sgiVmDuEgycR7YOlbX6OsVUHrUlpwhY3hgfMe6UtkMvhBvHF/WhroBEIJm1pV
PXZ/CbMCgYEApNWVktFBjOaYfY6SNn4iSts1jgsQbbpglg3kT7PLKjCAhI6lNsbk
dyT7ut01PL6RaW4SeQWtrJIVQaM6vF3pprMKqlc5XihOGAmVqH7rQx9rtQB5TicL
BFrwkQE4HQtQBV60hYQUzzlSk44VFDz+jxIEtacRHaomDRh2FtOTz+I=
-----END RSA PRIVATE KEY-----
`
	certFile, err := ioutil.TempFile("", "cert")
	assert.Nil(t, err)
	defer os.Remove(certFile.Name())
	certFile.Write([]byte(cert))
	certFile.Close()
	keyFile, err := ioutil.TempFile("", "key")
	assert.Nil(t, err)
	defer os.Remove(keyFile.Name())
	keyFile.Write([]byte(key))
	keyFile.Close()

	libkv.AddStore("mock", libkvmock.New)
	d := &Discovery{backend: "mock"}
	err = d.Initialize("127.0.0.3:1234", 0, 0, map[string]string{
		"kv.cacertfile": certFile.Name(),
		"kv.certfile":   certFile.Name(),
		"kv.keyfile":    keyFile.Name(),
	})
	assert.Nil(t, err)
	s := d.store.(*libkvmock.Mock)
	assert.Equal(t, s.Options.ClientTLS.CACertFile, certFile.Name())
	assert.Equal(t, s.Options.ClientTLS.CertFile, certFile.Name())
	assert.Equal(t, s.Options.ClientTLS.KeyFile, keyFile.Name())
}

func TestWatch(t *testing.T) {
	storeMock, err := libkvmock.New([]string{"127.0.0.1:1234"}, nil)
	assert.NotNil(t, storeMock)
	assert.NoError(t, err)

	d := &Discovery{backend: store.CONSUL}
	d.Initialize("127.0.0.1:1234/path", 0, 0, nil)
	d.store = storeMock

	s := d.store.(*libkvmock.Mock)
	mockCh := make(chan []*store.KVPair)

	// The first watch will fail on those three calls
	s.On("Exists", "path/"+defaultDiscoveryPath).Return(false, errors.New("test error"))
	s.On("Put", "path/"+defaultDiscoveryPath, mock.Anything, mock.Anything).Return(errors.New("test error"))
	s.On("WatchTree", "path/"+defaultDiscoveryPath, mock.Anything).Return(mockCh, errors.New("test error")).Once()

	// The second one will succeed.
	s.On("WatchTree", "path/"+defaultDiscoveryPath, mock.Anything).Return(mockCh, nil).Once()
	expected := discovery.Entries{
		&discovery.Entry{Host: "1.1.1.1", Port: "1111"},
		&discovery.Entry{Host: "2.2.2.2", Port: "2222"},
	}
	kvs := []*store.KVPair{
		{Key: path.Join("path", defaultDiscoveryPath, "1.1.1.1"), Value: []byte("1.1.1.1:1111")},
		{Key: path.Join("path", defaultDiscoveryPath, "2.2.2.2"), Value: []byte("2.2.2.2:2222")},
	}

	stopCh := make(chan struct{})
	ch, errCh := d.Watch(stopCh)

	// It should fire an error since the first WatchTree call failed.
	assert.EqualError(t, <-errCh, "test error")
	// We have to drain the error channel otherwise Watch will get stuck.
	go func() {
		for range errCh {
		}
	}()

	// Push the entries into the store channel and make sure discovery emits.
	mockCh <- kvs
	assert.Equal(t, <-ch, expected)

	// Add a new entry.
	expected = append(expected, &discovery.Entry{Host: "3.3.3.3", Port: "3333"})
	kvs = append(kvs, &store.KVPair{Key: path.Join("path", defaultDiscoveryPath, "3.3.3.3"), Value: []byte("3.3.3.3:3333")})
	mockCh <- kvs
	assert.Equal(t, <-ch, expected)

	// Make sure that if an error occurs it retries.
	// This third call to WatchTree will be checked later by AssertExpectations.
	s.On("WatchTree", "path/"+defaultDiscoveryPath, mock.Anything).Return(mockCh, nil)
	close(mockCh)
	// Give it enough time to call WatchTree.
	time.Sleep(3)

	// Stop and make sure it closes all channels.
	close(stopCh)
	assert.Nil(t, <-ch)
	assert.Nil(t, <-errCh)

	s.AssertExpectations(t)
}
