/*
 * Kiloproxy is a high-performance Cryptonote Stratum mining proxy.
 * Copyright (C) 2023 Kilopool.com
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package stratumserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"kiloproxy/config"
	"kiloproxy/kilolog"
	"kiloproxy/mutex"
	"math/big"
	"net"
	"os"
	"strconv"
	"time"
)

type Server struct {
	Connections []*Connection
	ConnsMut    mutex.Mutex

	NewConnections chan *Connection
}

type Connection struct {
	Conn net.Conn
	Id   uint64

	Upstream uint64

	mutex.Mutex
}

func (c *Connection) Send(a any) error {
	data, err := json.Marshal(a)
	if err != nil {
		panic(err)
	}
	return c.SendBytes(data)
}
func (c *Connection) SendBytes(data []byte) error {
	c.Conn.SetWriteDeadline(time.Now().Add(config.WRITE_TIMEOUT_SECONDS * time.Second))
	_, err := c.Conn.Write(append(data, '\n'))
	if err != nil {
		return err
	}
	return nil
}

func randomUint64() uint64 {
	b := make([]byte, 8)
	rand.Read(b)

	return binary.BigEndian.Uint64(b)
}

// Returns certPem, keyPem, err
func GenCertificate() ([]byte, []byte, error) {
	/*key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return []byte{}, []byte{}, err
	}*/
	pubkey, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return []byte{}, []byte{}, err
	}

	keyBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return []byte{}, []byte{}, err
	}

	// PEM encoding of private key (key.pem)
	keyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "OPENSSH PRIVATE KEY",
			Bytes: keyBytes,
		},
	)

	notBefore := time.Now()
	notAfter := notBefore.Add(10 * 365 * 24 * time.Hour)

	//Create certificate templet
	template := x509.Certificate{
		SerialNumber:          big.NewInt(0),
		Subject:               pkix.Name{CommonName: "localhost"},
		SignatureAlgorithm:    x509.PureEd25519, //x509.SHA256WithRSA,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement | x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	//Create certificate using templet
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pubkey, key)
	if err != nil {
		return []byte{}, []byte{}, err

	}
	// PEM encoding of certificate (certificate.pem)
	certPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: derBytes,
		},
	)
	err = os.WriteFile("./key.pem", keyPem, 0o666)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	return certPem, keyPem, os.WriteFile("./certificate.pem", certPem, 0o666)
}

func (s *Server) Start(port uint16, bind string, isTls bool) {
	if s.NewConnections == nil {
		s.NewConnections = make(chan *Connection, 1)
	}

	var listener net.Listener
	var err error
	if isTls {
		cert, err := tls.LoadX509KeyPair("./certificate.pem", "key.pem")

		if err != nil {
			kilolog.Info("Failed to load TLS certificate from file, generating a new one.")
			kilolog.Debug(err)

			certPem, keyPem, err := GenCertificate()
			if err != nil {
				kilolog.Fatal(err)
			}

			cert, err = tls.X509KeyPair(certPem, keyPem)
			if err != nil {
				kilolog.Fatal(err)
			}
		}

		fingerprint := sha256.Sum256(cert.Certificate[0])

		kilolog.Info("TLS fingerprint (SHA-256):", hex.EncodeToString(fingerprint[:]))

		listener, err = tls.Listen("tcp", bind+":"+strconv.FormatUint(uint64(port), 10), &tls.Config{
			Certificates: []tls.Certificate{cert},
		})
	} else {
		listener, err = net.Listen("tcp", bind+":"+strconv.FormatUint(uint64(port), 10))
	}
	if err != nil {
		kilolog.Fatal(err)
	}

	kilolog.Info("Stratum server listening on", fmt.Sprintf("%s:%d", bind, port))

	for {
		c, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		kilolog.Info("New incoming connection:", c.RemoteAddr().String())

		conn := &Connection{
			Conn: c,
			Id:   randomUint64(),
		}
		go s.handleConnection(conn)
	}
}

func (srv *Server) handleConnection(conn *Connection) {
	srv.ConnsMut.Lock()
	srv.Connections = append(srv.Connections, conn)
	srv.ConnsMut.Unlock()

	srv.NewConnections <- conn
}
