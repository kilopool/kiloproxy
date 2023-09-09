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
	"encoding/json"
	"kiloproxy/stratum/template"
)

type MinedShare struct {
	RequestID uint64
	ID        string
	JobID     string
	Nonce     string
	Result    string
}
type JobResultRequest struct {
	ID     string `json:"id"`
	JobID  string `json:"job_id"`
	Nonce  string `json:"nonce"`
	Result string `json:"result"`
}

type RequestGeneric struct {
	ID     uint64 `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}
type RequestJob struct {
	ID     uint64           `json:"id"`
	Method string           `json:"method"`
	Params JobResultRequest `json:"params"`
}
type RequestLogin struct {
	ID     uint64   `json:"id"`
	Method string   `json:"method"`
	Params loginReq `json:"params"`
}

type loginReq struct {
	Login           string   `json:"login"`
	Pass            string   `json:"pass"`
	Agent           string   `json:"agent"`
	Algo            []string `json:"algo"`
	NicehashSupport bool     `json:"nicehash_support"` // Non-standard. Not supported by XMRIG.
}
type Response struct {
	ID      uint64 `json:"id"`
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`

	Params any              `json:"params"` // used to send jobs over the connection
	Result *json.RawMessage `json:"result"` // used to return SubmitWork

	Error *ErrorJson `json:"error,omitempty"`
}

type LoginResponse struct {
	ID     uint64 `json:"id"`
	Status string
	Result LoginResponseResult `json:"result"`
	Error  *ErrorJson          `json:"error"`
}
type LoginResponseResult struct {
	ID         string       `json:"id"`
	Job        template.Job `json:"job"`
	Extensions []string     `json:"extensions"`
	Status     string       `json:"status"`
}

type LoginResponseJob struct {
	Algo     string `json:"algo"`
	Blob     string `json:"blob"`   // The blockhashing blob
	Height   uint64 `json:"height"` // only used in RandomX jobs
	JobID    string `json:"job_id"`
	SeedHash string `json:"seed_hash"` // only used in RandomX jobs
	Target   string `json:"target"`
	Id       string `json:"id"`
}

/*
	type LoginResponse struct {
		ID     uint64      `json:"id"`
		Status string      `json:"status"`
		Result LoginResult `json:"result"`
		Error  *ErrorJson  `json:"error,omitempty"`
	}

	type LoginResult struct {
		Job        *template.Job `json:"job"`
		ID         string        `json:"id"`
		Status     string        `json:"status"`
		Extensions []string      `json:"extensions,omitempty"`
	}
*/
type ErrorJson struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Reply struct {
	ID      uint64     `json:"id"`
	Jsonrpc string     `json:"jsonrpc"`
	Error   *ErrorJson `json:"error"`
	Result  any        `json:"result,omitempty"`
}

type JobNotification struct {
	Jsonrpc string       `json:"jsonrpc"`
	Method  string       `json:"method"`
	Params  template.Job `json:"params"`
}
