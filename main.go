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
	"encoding/json"
	"fmt"
	"kiloproxy/config"
	"kiloproxy/kilolog"
	"kiloproxy/stats"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	err := loadConfig()
	if err != nil {
		kilolog.Info(fmt.Sprintf("Failed to read config.json (%s), running configurator", err))
		configurator()
	}
	err = config.CFG.Validate()
	if err != nil {
		kilolog.Fatal(err)
	}

	kilolog.StartLogger()

	threads := runtime.GOMAXPROCS(0)
	if threads > config.CFG.MaxConcurrency {
		threads = config.CFG.MaxConcurrency
		runtime.GOMAXPROCS(config.CFG.MaxConcurrency)
	}

	if config.CFG.Title {
		colCyan := kilolog.COLOR_CYAN
		colGreen := kilolog.COLOR_GREEN
		colWhite := kilolog.COLOR_WHITE
		bold := kilolog.BOLD

		numThreads := strconv.FormatInt(int64(threads), 10)
		threadsCol := kilolog.COLOR_GREEN

		if numThreads == "2" {
			threadsCol = kilolog.COLOR_YELLOW
		} else if numThreads == "1" {
			threadsCol = kilolog.COLOR_RED
		}

		hasCgo := "cgo"
		if !stats.CGO {
			hasCgo = ""
		}

		kilolog.Printf("%s * %s%s\n", bold+colGreen, colWhite,
			"VERSION      "+colCyan+"Kiloproxy"+colWhite+" v"+config.VERSION.ToString())
		kilolog.Printf("%s * %s%s\n", bold+colGreen, colWhite,
			"CREDITS      "+colCyan+"Developed by "+colWhite+"Kilopool.com"+colCyan+".")
		kilolog.Printf("%s * %s%s\n", bold+colGreen, colWhite,
			"PLATFORM     "+runtime.GOOS+"/"+runtime.GOARCH+" "+kilolog.COLOR_CYAN+hasCgo)
		kilolog.Printf("%s * %s%s\n", bold+colGreen, colWhite,
			"CONCURRENCY  "+threadsCol+numThreads+colWhite+" threads")

		for i, v := range config.CFG.Pools {
			col := colCyan
			if v.Tls {
				col = colGreen
			}

			kilolog.Printf("%s * %s%s\n", bold+colGreen, colWhite,
				fmt.Sprintf("POOL #%d      %s", i, col+v.Url+kilolog.COLOR_RESET))
		}

	}

	kilolog.Info("Using pool", config.CFG.Pools[0].Url)

	go Stats()

	StartProxy()
}

func loadConfig() error {
	data, err := os.ReadFile("./config.json")
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &config.CFG)
}

var wordRegexp = regexp.MustCompile("^\\w+$")
var xmrRegexp = regexp.MustCompile("^[48][0-9AB][1-9A-HJ-NP-Za-km-z]{93}$")
var zephRegexp = regexp.MustCompile("^ZEPH[1-9A-HJ-NP-Za-km-z]+$")

func configurator() {
	userAddr := prompt("Enter your wallet address: ")
	kilolog.Info(userAddr)

	addr := []byte(userAddr)

	if !wordRegexp.Match([]byte(addr)) {
		kilolog.Fatal("Invalid address", addr)
	}
	curcfg := strings.ReplaceAll(string(config.DefaultConfig), "YOUR_WALLET_ADDRESS", userAddr)

	if xmrRegexp.Match(addr) {
		curcfg = strings.ReplaceAll(curcfg, "PORT_TLS", "3334")
		curcfg = strings.ReplaceAll(curcfg, "PORT_NO_TLS", "3333")
	} else if zephRegexp.Match(addr) {
		curcfg = strings.ReplaceAll(curcfg, "PORT_TLS", "5556")
		curcfg = strings.ReplaceAll(curcfg, "PORT_NO_TLS", "5555")
	} else {
		curcfg = strings.ReplaceAll(curcfg, "PORT_TLS", "3334")
		curcfg = strings.ReplaceAll(curcfg, "PORT_NO_TLS", "3333")
	}

	os.WriteFile("./config.json", []byte(curcfg), 0o666)
	err := json.Unmarshal([]byte(curcfg), &config.CFG)
	if err != nil {
		kilolog.Fatal(err)
	}
}

func prompt(lbl string) string {
	var str string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(lbl)
		str, _ = r.ReadString('\n')
		if str != "" {
			break
		}
	}
	return strings.TrimSpace(str)
}
