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

package main

import (
	"bufio"
	"encoding/hex"
	"kiloproxy/config"
	"kiloproxy/kilolog"
	"kiloproxy/stratum/rpc"
	stratumserver "kiloproxy/stratum/server"
	"kiloproxy/stratum/template"
	"time"
)

var srv = stratumserver.Server{}

func StartProxy() {
	go func() {
		for {
			newConn := <-srv.NewConnections
			go HandleConnection(newConn)
		}
	}()

	for i, v := range config.CFG.Bind {
		if i != len(config.CFG.Bind)-1 {
			go srv.Start(v.Port, v.Host, v.Tls)
		} else {
			srv.Start(v.Port, v.Host, v.Tls)
		}
	}

}

func HandleConnection(conn *stratumserver.Connection) {
	// Read the login request
	req := stratumserver.RequestLogin{}
	conn.Conn.SetReadDeadline(time.Now().Add(config.WRITE_TIMEOUT_SECONDS * time.Second))
	reader := bufio.NewReaderSize(conn.Conn, config.MAX_REQUEST_SIZE)
	err := rpc.ReadJSON(&req, reader)
	if err != nil {
		kilolog.Debug("ReadJSON failed in server:", err)
		Kick(conn.Id)
		return
	}
	reqParams := req.Params
	if reqParams.Agent == "" || reqParams.Login == "" || reqParams.Pass == "" {
		kilolog.Debug("client sent a malformed login request")
		Kick(conn.Id)
		return
	}

	kilolog.Debug("Stratum server received connection")
	kilolog.Debug("login", reqParams.Login)
	kilolog.Debug("pass ", reqParams.Pass)
	kilolog.Debug("algo ", reqParams.Algo)
	kilolog.Debug("agent", reqParams.Agent)

	if reqParams.NicehashSupport {
		kilolog.Debug("Client supports Nicehash mode (nicehash_support is true)")
	}

	// Write login response

	conn.Lock()
	UpstreamsMut.Lock()
	jobData, clientId, upstreamId, err := GetJob(conn)
	UpstreamsMut.Unlock()
	if err != nil {
		kilolog.Warn(err)
		Kick(conn.Id)
		conn.Unlock()
		return
	}

	conn.Upstream = upstreamId

	loginResponse := stratumserver.LoginResponse{
		ID:     req.ID,
		Status: "OK",
		Result: stratumserver.LoginResponseResult{
			ID: clientId,
			Job: template.Job{
				Algo:     jobData.Algo,
				Blob:     jobData.Blob,
				Height:   jobData.Height,
				JobID:    jobData.JobID,
				SeedHash: jobData.SeedHash,
				Target:   jobData.Target,
			},
			Status:     "OK",
			Extensions: []string{"keepalive", "nicehash"},
		},
		Error: nil,
	}
	conn.Send(loginResponse)
	conn.Unlock()

	// Listen for submitted shares

	for {
		req := stratumserver.RequestJob{}
		conn.Conn.SetReadDeadline(time.Now().Add(time.Duration(config.READ_TIMEOUT_SECONDS) * time.Second))
		reader := bufio.NewReaderSize(conn.Conn, config.MAX_REQUEST_SIZE)
		err := rpc.ReadJSON(&req, reader)

		if err != nil {
			kilolog.Debug("conn.go ReadJSON failed in server:", err)
			Kick(conn.Id)
			return
		}

		if req.Method == "keepalived" {
			conn.Send(stratumserver.Reply{
				ID:      req.ID,
				Jsonrpc: "2.0",
				Result: map[string]any{
					"status": "KEEPALIVED",
				},
			})
			continue
		} else if req.Method != "submit" {
			kilolog.Warn("Unknown method", req.Method, ". Skipping.")
			continue
		}

		UpstreamsMut.Lock()
		if Upstreams[conn.Upstream] == nil {
			panic("Upstreams[conn.Upstream] is nil")
		}

		var diff uint64
		if len(Upstreams[conn.Upstream].LastJob.Target) == 16 {
			dec, err := hex.DecodeString(Upstreams[conn.Upstream].LastJob.Target)
			if err != nil {
				kilolog.Err(err)
				Kick(conn.Id)
				return
			}
			diff = template.MidDiffToDiff(dec)
		} else {
			dec, err := hex.DecodeString(Upstreams[conn.Upstream].LastJob.Target)
			if err != nil {
				kilolog.Err(err)
				Kick(conn.Id)
				return
			}
			diff = template.ShortDiffToDiff(dec)
		}

		foundShares = append(foundShares, FoundShare{
			Time: time.Now(),
			Diff: diff,
		})

		res, err := Upstreams[conn.Upstream].Stratum.SubmitWork(req.Params.Nonce, req.Params.JobID, req.Params.Result, req.ID)
		UpstreamsMut.Unlock()
		if err != nil {
			kilolog.Err(err)
			Kick(conn.Id)
			return
		} else if res == nil {
			kilolog.Err("response is nil")
			Kick(conn.Id)
			return
		}

		kilolog.Debug("Sending SubmitWork response to client", res)

		conn.Send(res)
	}
}

// Note: srv.ConnsMut must be locked before calling this
func Kick(id uint64) {
	for i, v := range srv.Connections {
		if v.Id == id {
			// Close the connection
			v.Conn.Close()

			if Upstreams[v.Upstream] != nil {
				// remove client from upstream
				UpstreamsMut.Lock()
				for clid, clval := range Upstreams[v.Upstream].Clients {
					if clval == v.Id {
						Upstreams[v.Upstream].Clients[clid] = Upstreams[v.Upstream].Clients[len(Upstreams[v.Upstream].Clients)-1]
						Upstreams[v.Upstream].Clients = Upstreams[v.Upstream].Clients[:len(Upstreams[v.Upstream].Clients)-1]
					}
				}
				// If upstream is empty, close it
				if len(Upstreams[v.Upstream].Clients) == 0 {
					Upstreams[v.Upstream].Close()
				}
				UpstreamsMut.Unlock()
			}

			// remove client from server connections
			if len(srv.Connections) > 1 {
				srv.Connections = append(srv.Connections[:i], srv.Connections[i+1:]...)
			} else {
				srv.Connections = make([]*stratumserver.Connection, 0, 100)
			}
		}
	}
}

func GetNewJob(conn *stratumserver.Connection, job rpc.CompleteJob) {
	conn.Lock()
	defer conn.Unlock()

	jobData, _, upstreamId, err := GetJob(conn)
	if err != nil {
		kilolog.Warn(err)
		Kick(conn.Id)
		return
	}
	if conn.Upstream != upstreamId {
		kilolog.Debug("Upstream changed:", conn.Upstream)
		conn.Upstream = upstreamId
	}

	jobContent := rpc.JobRpc{
		Jsonrpc: "2.0",
		Method:  "job",

		Params: jobData,
	}

	err = conn.Send(jobContent)
	if err != nil {
		kilolog.Err(err)
	}
}
