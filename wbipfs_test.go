package wbipfs

import (
	"context"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestWayback(t *testing.T) {
	link := "https://example.com/"
	wbrc := &Archiver{
		Timeout:  30 * time.Second,
		IPFSHost: "localhost",
		IPFSPort: 5001,
		// UseTor:   true,
	}
	input, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}

	_, err = wbrc.Wayback(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransfer(t *testing.T) {
	content := []byte("Hello, IPFS!")
	tmpfile, err := ioutil.TempFile("", "wbipfs-testing")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}

	tr := NewDaemon("localhost", 5001)
	cid, err := tr.Transfer(tmpfile.Name())
	if err != nil {
		t.Error(err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	t.Log(cid)
}

func TestPinner(t *testing.T) {
	content := []byte("Hello, IPFS!")
	tmpfile, err := ioutil.TempFile("", "wbipfs-testing")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}

	cid, err := Pinner(tmpfile.Name())
	if err != nil {
		t.Error(err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	t.Log(cid)
}
