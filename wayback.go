package wbipfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/go-shiori/obelisk"
)

// Archiver is core of the wbipfs.
type Archiver struct {
	UseTor  bool
	Timeout time.Duration
	context func(ctx context.Context, network, addr string) (net.Conn, error)

	IPFSHost string
	IPFSPort uint
	IPFSMode string // daemon and pinner, default: pinner
}

// Wayback is the handle of saving to IPFS.
func (wbrc *Archiver) Wayback(links []string) (map[string]string, error) {
	var worklist = make(map[string]string)

	// Write content to tmp file
	dir, err := ioutil.TempDir(os.TempDir(), "wbipfs-")
	if err != nil {
		return worklist, fmt.Errorf("Create temp directory failed: %w", err)
	}
	defer os.RemoveAll(dir)

	if wbrc.UseTor {
		if err, tor := wbrc.dial(); err != nil {
			return worklist, fmt.Errorf("Dial tor failed: %w", err)
		} else {
			defer tor.Close()
		}
	}
	if wbrc.IPFSMode == "" {
		wbrc.IPFSMode = "pinner"
	}

	wg := sync.WaitGroup{}
	worker := NewDaemon(wbrc.IPFSHost, wbrc.IPFSPort)
	for idx, link := range links {
		idx, link := idx, link
		if err := isURL(link); err != nil {
			worklist[link] = fmt.Sprint(err)
			continue
		}
		wg.Add(1)

		go func(link string) {
			defer wg.Done()

			req := obelisk.Request{URL: link}
			arc := &obelisk.Archiver{
				EnableLog:   false,
				DialContext: wbrc.context,
				DisableJS:   disableJS(link),

				SkipResourceURLError: true,
			}
			arc.Validate()

			content, _, err := arc.Archive(context.Background(), req)
			if err != nil {
				log.Printf("Archive failed: %s", err)
				worklist[link] = "Archive failed."
				return
			}

			filepath := filepath.Join(dir, fmt.Sprintf("page-%d.html", idx))
			if err := ioutil.WriteFile(filepath, content, 0666); err != nil {
				log.Printf("Write failed, path: %s, err: %s", filepath, err)
				worklist[link] = "Archive failed."
				return
			}

			switch wbrc.IPFSMode {
			case "daemon":
				// Valid IPFS daemon connection
				if wbrc.IPFSHost == "" || wbrc.IPFSPort == 0 || wbrc.IPFSPort > 65535 {
					log.Printf("IPFS hostname or port is invalid, host: %s, port: %d", wbrc.IPFSHost, wbrc.IPFSPort)
					return
				}
				cid, err := worker.Transfer(filepath)
				if err != nil {
					log.Printf("Transfer failed, path: %s, err: %s", filepath, err)
					worklist[link] = "Archive failed."
					return
				}
				dest := "https://ipfs.io/ipfs/" + cid
				worklist[link] = dest
			case "pinner":
				if cid, err := Pinner(filepath); err != nil {
					log.Printf("Pin failed, path: %s, err: %s", filepath, err)
					worklist[link] = "Archive failed."
					return
				} else {
					dest := "https://ipfs.io/ipfs/" + cid
					worklist[link] = dest
				}
			}
		}(link)
	}
	wg.Wait()

	return worklist, nil
}

func isURL(link string) error {
	if link == "" {
		return fmt.Errorf("is not specified")
	}

	u, err := url.ParseRequestURI(link)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return fmt.Errorf("is not valid")
	}

	return nil
}

func (wbrc *Archiver) dial() (error, *tor.Tor) {
	// Lookup tor executable file
	if _, err := exec.LookPath("tor"); err != nil {
		return fmt.Errorf("%w", err), nil
	}

	// Start tor with default config
	t, err := tor.Start(nil, nil)
	if err != nil {
		return fmt.Errorf("Make connection failed: %w", err), nil
	}
	// defer t.Close()

	// Wait at most a minute to start network and get
	dialCtx, dialCancel := context.WithTimeout(context.Background(), time.Minute)
	defer dialCancel()

	// Make connection
	dialer, err := t.Dialer(dialCtx, nil)
	if err != nil {
		t.Close()
		return fmt.Errorf("Make connection failed: %w", err), nil
	}

	wbrc.context = dialer.DialContext

	return nil, t
}

func disableJS(link string) bool {
	// e.g. DISABLEJS_URIS=wikipedia.org|eff.org/tags
	uris := os.Getenv("DISABLEJS_URIS")
	if uris == "" {
		return false
	}

	regex := regexp.QuoteMeta(strings.ReplaceAll(uris, "|", "@@"))
	re := regexp.MustCompile(`(?m)` + strings.ReplaceAll(regex, "@@", "|"))

	return re.MatchString(link)
}
