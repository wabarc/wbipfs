package wbipfs

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestWayback(t *testing.T) {
	var (
		links []string
		got   map[string]string
	)
	wbrc := &Archiver{
		Timeout:  30 * time.Second,
		IPFSHost: "localhost",
		IPFSPort: 5001,
		// UseTor:   true,
	}
	got, _ = wbrc.Wayback(links)
	if len(got) != 0 {
		t.Errorf("got = %d; want 0", len(got))
	}

	links = []string{"https://example.org/", "https://example.com/"}
	got, _ = wbrc.Wayback(links)
	if len(got) == 0 {
		t.Errorf("got = %d; want not equal 0", len(got))
	}

	for orig, dest := range got {
		t.Log(orig, "=>", dest)
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
