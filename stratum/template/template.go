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

package template

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"kiloproxy/kilolog"
	"math/big"
	"strconv"
)

type Template struct {
	Blob           []byte
	Difficulty     uint64
	Height         uint64
	ReservedOffset int
	SeedHash       string
}
type Job struct {
	Algo     string `json:"algo"`
	Blob     string `json:"blob"`   // The blockhashing blob
	Height   uint64 `json:"height"` // only used in RandomX jobs
	JobID    string `json:"job_id"`
	SeedHash string `json:"seed_hash"` // only used in RandomX jobs
	Target   string `json:"target"`
}

// Converts an uint64 diff to a 4-bytes target used by xmrig
func DiffToShortTarget(d uint64) string {
	var bigInt64 = big.NewInt(0xffffffff)
	if d == 0 {
		return "ffffffff"
	}
	minerDiff, _ := big.NewInt(0).SetString(strconv.FormatUint(d, 16), 16)
	bigInt := big.NewInt(0).Div(bigInt64, minerDiff)
	buf := bigInt.Bytes()
	reverse(buf)
	for len(buf) < 4 {
		buf = append(buf, 0)
	}
	return hex.EncodeToString(buf)
}

// Converts an uint64 difficulty to an 8-bytes target.
// Result hash must be < target to be valid
func DiffToTarget(d uint64) []byte {
	var bigInt64, _ = big.NewInt(0).SetString("ffffffffffffffff", 16)
	if d == 0 {
		return []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	}
	minerDiff, _ := big.NewInt(0).SetString(strconv.FormatUint(d, 16), 16)
	bigInt := big.NewInt(0).Div(bigInt64, minerDiff)
	buf := bigInt.Bytes()
	reverse(buf)
	for len(buf) < 4 {
		buf = append(buf, 0)
	}
	return buf
}

func reverse(b []byte) {
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
}
func reverse2(b []byte) (c []byte) {
	c = make([]byte, len(b))
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		c[i], c[j] = b[j], b[i]
	}
	return
}

var maxTarget big.Int

const shortDiffTarget uint64 = 0xffffffff
const midDiffTarget uint64 = 0xffffffffffffffff

func init() {
	maxTarget.SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF", 16)
}
func HashToDiff(hash []byte) uint64 {
	var diff = big.NewInt(0).SetBytes(reverse2(hash[:]))
	if diff.Int64() == 0 {
		return 0
	}
	diff.Div(&maxTarget, diff)
	if diff.IsUint64() {
		return diff.Uint64()
	}
	panic(fmt.Sprintf("THIS SHOULD NEVER HAPPEN!!!!! diff is %d", diff))
}

// Converts 4-byte short diff to uint64 diff
func ShortDiffToDiff(shortDiff []byte) uint64 {
	if len(shortDiff) != 4 {
		kilolog.Fatal("short diff length is not 4:", hex.EncodeToString(shortDiff))
	}
	var diff = uint64(binary.LittleEndian.Uint32(shortDiff[:]))
	if diff == 0 {
		return 0
	}
	diff = shortDiffTarget / diff
	return diff
}

// Converts 8-byte short diff to uint64 diff
func MidDiffToDiff(midDiff []byte) uint64 {
	if len(midDiff) != 8 {
		kilolog.Fatal("mid diff length is not 8:", hex.EncodeToString(midDiff))
	}
	var diff = binary.LittleEndian.Uint64(midDiff[:])
	if diff == 0 {
		return 0
	}
	diff = midDiffTarget / diff
	return diff
}
