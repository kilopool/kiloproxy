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
	"kiloproxy/config"
	"kiloproxy/kilolog"
	"strconv"
	"time"
)

var numMiners, numUpstreams int
var avgHashrate float64

type FoundShare struct {
	Time time.Time
	Diff uint64
}

var foundShares = make([]FoundShare, 10)

func formatHashrate(f float64) string {
	if f > 1000*1000 {
		return strconv.FormatFloat(f/1000/1000, 'f', 1, 64) + " M"
	} else if f > 1000 {
		return strconv.FormatFloat(f/1000, 'f', 1, 64) + " k"
	} else {
		return strconv.FormatFloat(f, 'f', 0, 64) + " "
	}
}

func Stats() {
	for {
		getStats()
		kilolog.Statsf("%s avg, miners: "+kilolog.COLOR_CYAN+"%d"+kilolog.COLOR_WHITE+", upstreams: "+kilolog.COLOR_CYAN+"%d"+kilolog.COLOR_WHITE,
			kilolog.COLOR_CYAN+formatHashrate(avgHashrate)+"H/s"+kilolog.COLOR_WHITE,
			numMiners,
			numUpstreams,
		)
		time.Sleep(time.Duration(config.CFG.PrintInterval) * time.Second)
	}
}

func getStats() {
	shares2 := make([]FoundShare, 0, len(foundShares))
	var totalDiff float64

	for _, v := range foundShares {
		if time.Since(v.Time) <= config.HASHRATE_AVG_MINUTES*time.Minute {
			shares2 = append(shares2, v)
			totalDiff += float64(v.Diff)
		}
	}
	foundShares = shares2

	avgHashrate = totalDiff / (config.HASHRATE_AVG_MINUTES * 60)

	srv.ConnsMut.Lock()
	numMiners = len(srv.Connections)
	srv.ConnsMut.Unlock()

	UpstreamsMut.Lock()
	numUpstreams = len(Upstreams)
	UpstreamsMut.Unlock()
}
