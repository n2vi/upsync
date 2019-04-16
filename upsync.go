/* Copyright 2019 Eric Grosse grosse@gmail.com.
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"upspin.io/client"
	"upspin.io/config"
	"upspin.io/transports"
	"upspin.io/upspin"
)

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

var lastUpsync int64 // Unix time when upsync was last completed

func main() {
	wdfi, err := os.Stat(".")
	chk(err)
	lastUpsync = wdfi.ModTime().Unix()

	// find first component of current directory that looks like email address
	// then make wd == upspin working directory
	wd, err := os.Getwd()
	chk(err)
	i := strings.IndexByte(wd, '@')
	if i < 0 {
		panic("couldn't find upspin user name in working directory")
	}
	i = strings.LastIndexAny(wd[:i], "\\/")
	if i < 0 {
		panic("unable to parse working directory")
	}
	slash := wd[i : i+1]
	wd = wd[i+1:]
	if slash != "/" {
		wd = strings.ReplaceAll(wd, slash, "/")
	}
	fmt.Println("working directory", wd)

	// initialize upspin client
	home := config.Home()
	cfg, err := config.FromFile(filepath.Join(home, "upspin", "config"))
	chk(err)
	transports.Init(cfg)
	upc := client.New(cfg)

	// start copying
	upsync(upc, wd, "")

	// try to set mtime of local dir as time of this upsync
	now := time.Now()
	_ = os.Chtimes(".", now, now)
}

func upsync(upc upspin.Client, wd, subdir string) {

	udir, err := upc.Glob(wd + "/" + subdir + "*")
	chk(err)
	ldir, err := ioutil.ReadDir(subdir + ".")
	chk(err)

	// compare sorted lists udir and ldir
	uj := 0
	lj := 0
	for {
		cmp := 0 // -1,0,1 as udir[uj] sorts before,same,after ldir[lj]
		if lj < len(ldir) && ldir[lj].Mode()&os.ModeSymlink != 0 {
			panic("local symlink not allowed! " + ldir[lj].Name())
		}
		if uj >= len(udir) {
			if lj >= len(ldir) {
				break // both lists exhausted
			} else {
				cmp = 1
			}
		} else if lj >= len(ldir) {
			cmp = -1
		} else {
			cmp = strings.Compare(string(udir[uj].SignedName)[len(wd)+1:], subdir+ldir[lj].Name())
		}

		// copy newer to older/missing
		switch cmp {
		case -1:
			pathname := string(udir[uj].SignedName)[len(wd)+1:]
			if udir[uj].Attr&upspin.AttrDirectory != 0 {
				err = os.Mkdir(pathname, 0700)
				chk(err)
				upsync(upc, wd, pathname+"/")
			} else if udir[uj].Attr&upspin.AttrLink != 0 {
				fmt.Println("ignoring upspin symlink", pathname)
			} else if udir[uj].Attr&upspin.AttrIncomplete != 0 {
				// as indicator of permission issue, create mode 0 length 0 placeholder
				empty := make([]byte, 0)
				err = ioutil.WriteFile(pathname, empty, 0)
				chk(err)
			} else {
				if len(udir[uj].Blocks) > 50 {
					fmt.Println("skipping big", pathname)
				} else {
					utime := int64(udir[uj].Time)
					pull(upc, wd, pathname, utime)
				}
			}
			uj++
		case 0:
			pathname := subdir + ldir[lj].Name()
			uIsDir := udir[uj].Attr&upspin.AttrDirectory != 0
			lIsDir := ldir[lj].IsDir()
			if uIsDir != lIsDir {
				panic("same name, different Directory attribute! " + pathname)
			}
			if uIsDir {
				upsync(upc, wd, pathname+"/")
			} else {
				utime := int64(udir[uj].Time)
				ltime := ldir[lj].ModTime().Unix()
				if utime > ltime {
					pull(upc, wd, pathname, utime)
				} else if utime < ltime {
					push(upc, wd, pathname, ltime)
				} // else assumed already in sync
			}
			uj++
			lj++
		case 1:
			pathname := subdir + ldir[lj].Name()
			if ldir[lj].IsDir() {
				fmt.Println("upspin mkdir", wd+"/"+pathname)
				_, err = upc.MakeDirectory(upspin.PathName(wd + "/" + pathname))
				chk(err)
				upsync(upc, wd, pathname+"/")
			} else {
				ltime := ldir[lj].ModTime().Unix()
				push(upc, wd, pathname, ltime)
			}
			lj++
		}
	}
}

func pull(upc upspin.Client, wd, pathname string, utime int64) {
	fmt.Println("pull", pathname)
	bytes, err := upc.Get(upspin.PathName(wd + "/" + pathname))
	chk(err)
	err = ioutil.WriteFile(pathname, bytes, 0600)
	chk(err)
	mtime := time.Unix(utime, 0)
	err = os.Chtimes(pathname, mtime, mtime)
	chk(err)
}

func push(upc upspin.Client, wd, pathname string, ltime int64) {
	if ltime < lastUpsync {
		fmt.Println("skipping old", pathname)
		return
	}
	fmt.Println("push", pathname)
	bytes, err := ioutil.ReadFile(pathname)
	chk(err)
	path := upspin.PathName(wd + "/" + pathname)
	_, err = upc.Put(path, bytes)
	chk(err)
	err = upc.SetTime(path, upspin.Time(ltime))
	chk(err)
}
