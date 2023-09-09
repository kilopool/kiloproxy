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

package config

import (
	"encoding/hex"
	"errors"
	"net"
)

var CFG Config

type Config struct {
	Pools []struct {
		Url            string `json:"url"`
		Tls            bool   `json:"tls"`
		TlsFingerprint string `json:"fingerprint"`
		User           string `json:"user"`
		Pass           string `json:"pass"`
	} `json:"pools"`
	Bind []struct {
		Host string `json:"host"`
		Port uint16 `json:"port"`
		Tls  bool   `json:"tls"`
	} `json:"bind"`
	PrintInterval  uint16 `json:"print_interval"`
	Interactive    bool   `json:"interactive"`
	MaxConcurrency int    `json:"max_concurrency"`
	Colors         bool   `json:"colors"`
	LogDate        bool   `json:"log_date"`
	Title          bool   `json:"title"`
	Verbose        bool   `json:"verbose"`
}

const DefaultConfig = `{
	"pools": [
		{
			"url": "eu.stratum.kilopool.com:PORT_TLS",
			"tls": true,
			"user": "YOUR_WALLET_ADDRESS",
			"pass": "x"
		},
		{
			"url": "eu.stratum.kilopool.com:PORT_NO_TLS",
			"tls": false,
			"user": "YOUR_WALLET_ADDRESS",
			"pass": "x"
		}
	],
	"bind": [
		{
			"host": "0.0.0.0",
			"port": 3333,
			"tls": false
		},
		{
			"host": "0.0.0.0",
			"port": 3334,
			"tls": true
		}
	],
	"print_interval": 60,
	"interactive": true,
	"max_concurrency": 4,
	"colors": true,
	"log_date": true,
	"title": true,
	"verbose": false
}`

func (c *Config) Validate() error {
	if len(c.Pools) == 0 {
		return errors.New("no pools defined")
	}
	for _, v := range c.Pools {
		if len(v.Url) == 0 {
			return errors.New("invalid pool url")
		}
		if v.TlsFingerprint != "" {
			if len(v.TlsFingerprint) != 64 {
				return errors.New("invalid SHA-256 TLS fingerprint length")
			}
			_, err := hex.DecodeString(v.TlsFingerprint)
			if err != nil {
				return errors.New("invalid SHA-256 TLS fingerprint")
			}
		}

	}

	if len(c.Bind) == 0 {
		return errors.New("bind is empty")
	}
	for _, v := range c.Bind {
		if len(v.Host) == 0 || net.ParseIP(v.Host) == nil {
			return errors.New("invalid bind host")
		}
		if v.Port == 0 {
			return errors.New("invalid bind port")
		}
	}
	if c.PrintInterval == 0 {
		return errors.New("invalid print interval")
	}
	if c.MaxConcurrency < 1 || c.MaxConcurrency > 128 {
		return errors.New("invalid max concurrency (should be between 1 and 128)")
	}
	return nil
}
