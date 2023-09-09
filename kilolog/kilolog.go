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

package kilolog

import (
	"fmt"
	"kiloproxy/config"
	"runtime"
	"strconv"
	"strings"
)

var debug = " DEBUG "
var info = " INFO  "
var warn = " WARN  "
var err = " ERR   "
var fatal = " FATAL "
var stats = " STATS "

var COLOR_RESET = "\x1b[0m"

var COLOR_BLACK = "\x1b[30m"
var COLOR_RED = "\x1b[31m"
var COLOR_GREEN = "\x1b[32m"
var COLOR_YELLOW = "\x1b[33m"
var COLOR_BLUE = "\x1b[34m"
var COLOR_MAGENTA = "\x1b[35m"
var COLOR_CYAN = "\x1b[36m"
var COLOR_WHITE = "\x1b[37m"

var BOLD = "\x1b[1m"
var FAINT = "\x1b[2m"

var COLOR_BG_BLACK = "\x1b[40m"
var COLOR_BG_RED = "\x1b[41m"
var COLOR_BG_GREEN = "\x1b[42m"
var COLOR_BG_YELLOW = "\x1b[43m"
var COLOR_BG_BLUE = "\x1b[44m"
var COLOR_BG_MAGENTA = "\x1b[45m"
var COLOR_BG_CYAN = "\x1b[36m"
var COLOR_BG_WHITE = "\x1b[47m"

func disableColors() {
	COLOR_RESET = ""
	COLOR_BLACK, COLOR_RED = "", ""
	COLOR_GREEN, COLOR_YELLOW = "", ""
	COLOR_BLUE, COLOR_MAGENTA = "", ""
	COLOR_CYAN, COLOR_WHITE = "", ""

	BOLD, FAINT = "", ""

	COLOR_BG_BLACK, COLOR_BG_RED = "", ""
	COLOR_BG_GREEN, COLOR_BG_YELLOW = "", ""
	COLOR_BG_BLUE, COLOR_BG_MAGENTA = "", ""
	COLOR_BG_CYAN, COLOR_BG_WHITE = "", ""
}

func StartLogger() {
	if config.CFG.Colors {
		debug = COLOR_BG_MAGENTA + BOLD + debug + COLOR_RESET + FAINT + " "
		info = COLOR_BG_BLUE + BOLD + info + COLOR_RESET + " "
		warn = COLOR_BG_YELLOW + BOLD + warn + COLOR_RESET + " "
		err = COLOR_BG_RED + BOLD + err + COLOR_RESET + " "
		fatal = COLOR_BG_RED + BOLD + fatal + COLOR_RESET + " "
		stats = COLOR_BG_GREEN + BOLD + stats + COLOR_RESET + " "
	} else {
		disableColors()
	}
}

func getCaller() string {
	_, file, line, _ := runtime.Caller(2)
	f := strings.Split(file, "/")
	out := strings.Split(f[len(f)-1], ".")[0] + ":" + strconv.FormatInt(int64(line), 10)
	for len(out) < 15 {
		out = out + " "
	}
	return out
}

func getPrefix() (out string) {
	if config.CFG.Verbose {
		out = getCaller()
	}

	return
}

func Debug(a ...any) {
	if !config.CFG.Verbose {
		return
	}

	fmt.Print(getPrefix() + debug + fmt.Sprintln(a...) + COLOR_RESET)
}
func Info(a ...any) {
	fmt.Print(getPrefix() + info + fmt.Sprintln(a...))
}
func Warn(a ...any) {
	fmt.Print(getPrefix() + warn + fmt.Sprintln(a...))
}
func Err(a ...any) {
	fmt.Print(getPrefix() + err + fmt.Sprintln(a...))
}
func Fatal(a ...any) {
	fmt.Print(getPrefix() + fatal + fmt.Sprintln(a...))
	panic(fmt.Sprintln(a...))
}
func Statsf(f string, a ...any) {
	fmt.Print(getPrefix() + stats + fmt.Sprintf(f, a...) + "\n")
}

func Printf(s string, a ...any) {
	fmt.Printf(s, a...)
}
