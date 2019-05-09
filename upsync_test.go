// Copyright 2019 The Upspin Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"upspin.io/bind"
	"upspin.io/client"
	"upspin.io/config"
	"upspin.io/factotum"
	"upspin.io/log"
	"upspin.io/test/testutil"
	"upspin.io/upspin"

	dirserver "upspin.io/dir/inprocess"
	keyserver "upspin.io/key/inprocess"
	storeserver "upspin.io/store/inprocess"
)

// adapted from upspinfs_test.go

var testConfig struct {
	syncDir string
	user    string
	cfg     upspin.Config
	upc     upspin.Client
}

const (
	perm           = 0777
	maxBytes int64 = 1e8
)

type testfile struct {
	path string
}

var testfiles = []testfile{
	{"foo"},
	{"bar/baz"},
}

// testSetup creates a temporary user config with inprocess services.
func testSetup(name string) (upspin.Config, error) {
	endpoint := upspin.Endpoint{
		Transport: upspin.InProcess,
		NetAddr:   "", // ignored
	}

	f, err := factotum.NewFromDir(testutil.Repo("key", "testdata", "user1")) // Always use user1's keys.
	if err != nil {
		return nil, err
	}

	cfg := config.New()
	cfg = config.SetUserName(cfg, upspin.UserName(name))
	cfg = config.SetPacking(cfg, upspin.EEPack)
	cfg = config.SetKeyEndpoint(cfg, endpoint)
	cfg = config.SetStoreEndpoint(cfg, endpoint)
	cfg = config.SetDirEndpoint(cfg, endpoint)
	cfg = config.SetFactotum(cfg, f)

	bind.RegisterKeyServer(upspin.InProcess, keyserver.New())
	bind.RegisterStoreServer(upspin.InProcess, storeserver.New())
	bind.RegisterDirServer(upspin.InProcess, dirserver.New(cfg))

	// publicKey := upspin.PublicKey(fmt.Sprintf("key for %s", name))
	publicKey := f.PublicKey()
	user := &upspin.User{
		Name:      upspin.UserName(name),
		Dirs:      []upspin.Endpoint{cfg.DirEndpoint()},
		Stores:    []upspin.Endpoint{cfg.StoreEndpoint()},
		PublicKey: publicKey,
	}
	key, err := bind.KeyServer(cfg, cfg.KeyEndpoint())
	if err != nil {
		return nil, err
	}
	err = key.Put(user)
	return cfg, err
}

func setup() error {
	// Set up a user config.
	testConfig.user = "tester@google.com"
	cfg, err := testSetup(testConfig.user)
	if err != nil {
		return err
	}
	testConfig.cfg = cfg

	// A directory for locally synced files.
	testConfig.syncDir, err = ioutil.TempDir("", "upsync")
	if err != nil {
		return err
	}
	err = os.Chdir(testConfig.syncDir)
	chk(err)

	log.SetLevel("info")

	// populate test files in upspin server
	upc := client.New(cfg)
	testConfig.upc = upc
	_, err = upc.MakeDirectory(upspin.PathName(testConfig.user))
	if err != nil {
		return err
	}
	_, err = upc.MakeDirectory(upspin.PathName(testConfig.user + "/bar"))
	if err != nil {
		return err
	}
	for _, f := range testfiles {
		path := upspin.PathName(testConfig.user + "/" + f.path)
		_, err := upc.Put(path, []byte(f.path))
		if err != nil {
			return err
		}
	}

	return nil
}

func cleanup() {
	os.RemoveAll(testConfig.syncDir)
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		fmt.Fprintf(os.Stderr, "setup failed: %s", err)
		cleanup()
		os.Exit(1)
	}
	rv := m.Run()
	cleanup()
	os.Exit(rv)
}

// TestSync test initial sync of test tree from server.
func TestSync(t *testing.T) {
	upsync(testConfig.upc, testConfig.user, "")
	fmt.Println(testConfig.syncDir, ":")
	fi, err := ioutil.ReadDir(testConfig.syncDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fi {
		fmt.Println(f.Name(), f.Size())
	}
	if len(fi) != len(testfiles) {
		t.Fatalf("expected %d files, saw %d\n", len(testfiles), len(fi))
	}
}
