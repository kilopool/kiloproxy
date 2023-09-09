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

// package stratumclient implements a Cryptonote Stratum mining protocol client
package stratumclient

import (
	"bufio"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"kiloproxy/config"
	"kiloproxy/kilolog"
	"kiloproxy/mutex"
	"kiloproxy/stratum/rpc"
	"net"
	"time"
)

const ()

type SubmitWorkResult struct {
	Status string `json:"status"`
}

type Client struct {
	destination     string
	conn            net.Conn
	responseChannel chan *rpc.Response

	ClientId string

	mutex mutex.Mutex

	alive bool
}

func (cl *Client) IsAlive() bool {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	return cl.alive
}

// Connect to the stratum server port with the given login info. Returns error if connection could
// not be established, or if the stratum server itself returned an error. In the latter case,
// code and message will also be specified. If the stratum server returned just a warning, then
// error will be nil, but code & message will be specified.
func (cl *Client) Connect(
	destination string, useTLS bool, tlsFingerprint string, agent,
	uname, pw string) (jobChan <-chan *rpc.CompleteJob, err error) {
	cl.Close()
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	cl.destination = destination

	if useTLS {
		cl.conn, err = tls.Dial("tcp", destination, &tls.Config{
			InsecureSkipVerify: true,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				computedFingerprint := sha256.Sum256(rawCerts[0])
				computedFingerprintStr := hex.EncodeToString(computedFingerprint[:])

				if tlsFingerprint == "" {
					kilolog.Info("Pool fingerprint", computedFingerprintStr)
					return nil
				}

				if len(rawCerts) != 1 {
					kilolog.Err("invalid number of certificates")
					return errors.New("invalid number of certificates")
				}

				if computedFingerprintStr != tlsFingerprint {
					kilolog.Err("invalid pool TLS fingerprint:", computedFingerprintStr, "expected", tlsFingerprint)
					return errors.New("invalid fingerprint")
				}

				return nil
			},
		})
	} else {
		cl.conn, err = net.DialTimeout("tcp", destination, time.Second*30)
	}

	if err != nil {
		kilolog.Warn("Connection failed:", err, cl)
		return nil, err
	}
	// send login
	loginRequest := &struct {
		ID     uint64 `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}{
		ID:     1,
		Method: "login",
		Params: struct {
			Login string `json:"login"`
			Pass  string `json:"pass"`
			Agent string `json:"agent"`
		}{
			Login: uname,
			Pass:  pw,
			Agent: agent,
		},
	}

	data, err := json.Marshal(loginRequest)
	if err != nil {
		kilolog.Debug("json marshalling failed:", err, "for client")
		return nil, err
	}
	cl.conn.SetWriteDeadline(time.Now().Add(config.WRITE_TIMEOUT_SECONDS * time.Second))

	kilolog.Debug("sending to pool:", string(data))

	data = append(data, '\n')
	if _, err = cl.conn.Write(data); err != nil {
		kilolog.Warn(err)
		return nil, err
	}

	// read the login response
	response := &rpc.LoginResponse{}
	cl.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	rdr := bufio.NewReaderSize(cl.conn, config.MAX_REQUEST_SIZE)
	err = rpc.ReadJSON(response, rdr)
	if err != nil {
		kilolog.Warn(err)
		return nil, err
	}
	if response.Result == nil {
		if response.Error != nil {
			kilolog.Warn("client login error:", response.Error)
			return nil, errors.New("stratum server error")
		}
		kilolog.Warn("malformed login response:", response)
		return nil, errors.New("malformed login response")
	}

	cl.responseChannel = make(chan *rpc.Response)
	cl.alive = true
	jc := make(chan *rpc.CompleteJob)
	if response.Result.Job == nil {
		kilolog.Warn("malformed login response: result:", response.Result)
		return nil, fmt.Errorf("malformed login response")
	}

	cl.ClientId = response.Result.ID

	go cl.dispatchJobs(cl.conn, jc, response.Result.Job, cl.responseChannel)
	return jc, nil
}

func (cl *Client) submitRequest(requestData any, expectedResponseID uint64) (*rpc.Response, error) {
	cl.mutex.Lock()
	if !cl.alive {
		cl.mutex.Unlock()
		return nil, errors.New("client is not alive")
	}
	data, err := json.Marshal(requestData)
	if err != nil {
		kilolog.Warn("failed to submit work:", err)
		cl.mutex.Unlock()
		return nil, err
	}
	cl.conn.SetWriteDeadline(time.Now().Add(config.WRITE_TIMEOUT_SECONDS * time.Second))
	kilolog.Debug("sending to pool:", string(data))
	data = append(data, '\n')
	if _, err = cl.conn.Write(data); err != nil {
		kilolog.Warn("failed to submit work:", err)
		cl.mutex.Unlock()
		return nil, err
	}
	respChan := cl.responseChannel
	cl.mutex.Unlock()

	// await the response
	response := <-respChan
	if response == nil {
		return nil, fmt.Errorf("failed to submit work: empty response")
	}
	if response.ID != expectedResponseID {
		kilolog.Warn("unexpected response ID: got:", response.ID, "expected:", expectedResponseID)

		return nil, fmt.Errorf("failed to submit work: unexpected response")
	}
	return response, nil
}

// If error is returned by this method, then client will be closed and put in not-alive state.
func (cl *Client) SubmitWork(nonce, jobid, result string, id uint64) (*rpc.Response, error) {
	submitRequest := &struct {
		ID     uint64      `json:"id"`
		Method string      `json:"method"`
		Params interface{} `json:"params"`
	}{
		ID:     id,
		Method: "submit",
		Params: &struct {
			ID     string `json:"id"`
			JobID  string `json:"job_id"`
			Nonce  string `json:"nonce"`
			Result string `json:"result"`
		}{cl.ClientId, jobid, nonce, result},
	}
	return cl.submitRequest(submitRequest, id)
}

func (cl *Client) Close() {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()
	if !cl.alive {
		return
	}
	cl.alive = false
	cl.conn.Close()
}

// dispatchJobs will forward incoming jobs to the JobChannel until error is received or the
// connection is closed. Client will be in not-alive state on return.
func (cl *Client) dispatchJobs(conn net.Conn, jobChan chan<- *rpc.CompleteJob, firstJob *rpc.CompleteJob, responseChan chan<- *rpc.Response) {
	defer func() {
		close(jobChan)
		close(responseChan)
	}()
	jobChan <- firstJob
	reader := bufio.NewReaderSize(conn, config.MAX_REQUEST_SIZE)
	for {
		response := &rpc.Response{}
		conn.SetReadDeadline(time.Now().Add(5 * 60 * time.Second))
		err := rpc.ReadJSON(response, reader)
		if err != nil {
			if cl.alive {
				kilolog.Warn("failed to read jobs from pool:", err)
				break
			} else {
				kilolog.Debug("failed to read jobs from pool:", err)
				break
			}
		}
		if response.Method != "job" {
			responseChan <- response
			continue
		}

		jobChan <- response.Job
	}
}
