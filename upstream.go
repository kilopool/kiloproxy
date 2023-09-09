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
	"encoding/hex"
	"errors"
	"fmt"
	"kiloproxy/config"
	"kiloproxy/kilolog"
	"kiloproxy/mutex"
	stratumclient "kiloproxy/stratum/client"
	"kiloproxy/stratum/rpc"
	stratumserver "kiloproxy/stratum/server"
)

type Upstream struct {
	Clients []uint64

	TopNicehash byte

	Stratum *stratumclient.Client

	ID uint64

	LastJob rpc.CompleteJob
}

var Upstreams = make(map[uint64]*Upstream, 100)
var UpstreamsMut mutex.Mutex
var LatestUpstream uint64

// GetJob returns a job from the Upstream, the client ID, and the upstream ID
func GetJob(conn *stratumserver.Connection) (rpc.CompleteJob, string, uint64, error) {
	connClientId := conn.Id

	var theJob rpc.CompleteJob
	var upstreamId uint64
	var nicehash byte

	if conn.Upstream != 0 {
		theJob = Upstreams[conn.Upstream].LastJob
		upstreamId = conn.Upstream
		nicehash = Upstreams[conn.Upstream].TopNicehash + 1
		Upstreams[LatestUpstream].TopNicehash++
	} else if len(Upstreams) == 0 || Upstreams[LatestUpstream].TopNicehash == 0xff {
		kilolog.Debug("New upstream connection")

		newId := LatestUpstream + 1
		client := &stratumclient.Client{}

		jobChan, err := client.Connect(
			config.CFG.Pools[0].Url,
			config.CFG.Pools[0].Tls,
			config.CFG.Pools[0].TlsFingerprint,
			config.USERAGENT,
			config.CFG.Pools[0].User,
			config.CFG.Pools[0].Pass,
		)
		if err != nil {
			return rpc.CompleteJob{}, "", 0, err
		}

		recvJob := <-jobChan

		if recvJob == nil {
			return rpc.CompleteJob{}, "", 0, errors.New("received nil job")
		}

		Upstreams[newId] = &Upstream{
			ID:          newId,
			Clients:     []uint64{connClientId},
			TopNicehash: 1,
			Stratum:     client,
			LastJob:     *recvJob,
		}
		LatestUpstream = newId

		go UpstreamHandler(Upstreams[newId], jobChan)

		theJob = *recvJob
		upstreamId = newId
		nicehash = 1
	} else {
		kilolog.Debug("Reusing upstream job")
		upstreamId = LatestUpstream
		theJob = Upstreams[upstreamId].LastJob
		nicehash = Upstreams[upstreamId].TopNicehash + 1
		Upstreams[upstreamId].TopNicehash++
		Upstreams[upstreamId].Clients = append(Upstreams[upstreamId].Clients, conn.Id)
	}
	kilolog.Debug("Nicehash byte is", hex.EncodeToString([]byte{nicehash}))

	blobBin, err := hex.DecodeString(theJob.Blob)
	if err != nil {
		return rpc.CompleteJob{}, "", 0, err
	}
	if len(blobBin) < 44 {
		return rpc.CompleteJob{}, "", 0, fmt.Errorf("mining blob is too short: %x", blobBin)
	}

	blobBin[42] = nicehash

	theJob.Blob = hex.EncodeToString(blobBin)

	return theJob, Upstreams[upstreamId].Stratum.ClientId, upstreamId, nil
}

func UpstreamHandler(us *Upstream, jobChan <-chan *rpc.CompleteJob) {
	for {
		recvJob := <-jobChan

		if recvJob == nil {
			if us.Stratum.IsAlive() {
				kilolog.Warn("recvJob is nil")
			} else {
				kilolog.Debug("recvJob is nil")
			}
			us.Close()
			return
		}

		kilolog.Debug("Received new job with Job ID", recvJob.JobID)

		HandleUpstreamJob(us, recvJob)
	}
}

func (us *Upstream) Close() {
	UpstreamsMut.Lock()
	defer UpstreamsMut.Unlock()

	us.Stratum.Close()

	Upstreams[us.ID] = nil

	if LatestUpstream == us.ID {
		if len(Upstreams) == 1 {
			kilolog.Debug("Last upstream destroyed.")
			LatestUpstream = 0
		}
	}

	delete(Upstreams, us.ID)
}

func HandleUpstreamJob(us *Upstream, job *rpc.CompleteJob) {
	kilolog.Debug("New job for Upstream", us.ID)

	UpstreamsMut.Lock()
	defer UpstreamsMut.Unlock()

	us.TopNicehash = 0

	us.LastJob = *job

	for _, v := range us.Clients {
		srv.ConnsMut.Lock()

		//innerloop:
		for _, conn := range srv.Connections {
			if conn.Id == v {
				kilolog.Debug("Refreshing job for connection", conn.Id)
				GetNewJob(conn, us.LastJob)
				//break innerloop
			}
		}

		srv.ConnsMut.Unlock()
	}
}
