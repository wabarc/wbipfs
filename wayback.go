package wbipfs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/go-shiori/obelisk"
	"github.com/pkg/errors"
)

// Archiver is core of the wbipfs.
type Archiver struct {
	UseTor      bool
	Timeout     time.Duration
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	IPFSHost string
	IPFSPort uint
	IPFSMode string // daemon and pinner, default: pinner
}

// Wayback is the handle of saving to IPFS.
func (wbrc *Archiver) Wayback(ctx context.Context, input *url.URL) (dst string, err error) {
	// Write content to tmp file
	dir, err := ioutil.TempDir(os.TempDir(), "wbipfs-")
	if err != nil {
		return dst, errors.Wrap(err, "create temp directory failed")
	}
	defer os.RemoveAll(dir)

	if wbrc.UseTor {
		if err, tor := wbrc.dial(ctx); err != nil {
			return dst, errors.Wrap(err, "dial tor failed")
		} else {
			defer tor.Close()
		}
	}
	if wbrc.IPFSMode == "" {
		wbrc.IPFSMode = "pinner"
	}

	uri := input.String()
	req := obelisk.Request{URL: uri, Input: inputFromContext(ctx)}
	arc := &obelisk.Archiver{
		EnableLog:   false,
		DialContext: wbrc.DialContext,
		DisableJS:   disableJS(uri),

		SkipResourceURLError: true,
	}
	arc.Validate()

	dst = "Archive failed."
	content, contentType, err := arc.Archive(ctx, req)
	if err != nil {
		log.Printf("Archive failed: %s", err)
		return dst, err
	}

	filepath := filepath.Join(dir, fileName(uri, contentType))
	if err := ioutil.WriteFile(filepath, content, 0666); err != nil {
		return dst, errors.Wrapf(err, "write failed, path: %s", filepath)
	}

	switch wbrc.IPFSMode {
	case "daemon":
		// Valid IPFS daemon connection
		if wbrc.IPFSHost == "" || wbrc.IPFSPort == 0 || wbrc.IPFSPort > 65535 {
			return dst, errors.Errorf("IPFS hostname or port is invalid, host: %s, port: %d", wbrc.IPFSHost, wbrc.IPFSPort)
		}
		worker := NewDaemon(wbrc.IPFSHost, wbrc.IPFSPort)
		cid, err := worker.Transfer(filepath)
		if err != nil {
			err = errors.Wrapf(err, "transfer failed, path: %s", filepath)
			break
		}
		dst = fmt.Sprintf("https://ipfs.io/ipfs/%s#%s", cid, uri)
	case "pinner":
		if cid, err := Pinner(filepath); err != nil {
			err = errors.Wrapf(err, "pin failed, path: %s", filepath)
			break
		} else {
			dst = fmt.Sprintf("https://ipfs.io/ipfs/%s", cid)
		}
	}

	return dst, nil
}

type ctxKeyInput struct{}

// ContextWithInput permits to inject a webpage into a context by given input.
func (wbrc *Archiver) ContextWithInput(ctx context.Context, input []byte) (c context.Context) {
	return context.WithValue(ctx, ctxKeyInput{}, input)
}

func inputFromContext(ctx context.Context) io.Reader {
	if b, ok := ctx.Value(ctxKeyInput{}).([]byte); ok {
		return bytes.NewReader(b)
	}
	return nil
}

func (wbrc *Archiver) dial(ctx context.Context) (error, *tor.Tor) {
	// Lookup tor executable file
	if _, err := exec.LookPath("tor"); err != nil {
		return err, nil
	}

	// Start tor with default config
	t, err := tor.Start(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "starts tor failed"), nil
	}
	// defer t.Close()

	// Wait at most a minute to start network and get
	dialCtx, dialCancel := context.WithTimeout(ctx, time.Minute)
	defer dialCancel()

	// Make connection
	dialer, err := t.Dialer(dialCtx, nil)
	if err != nil {
		t.Close()
		return errors.Wrap(err, "dial tor failed"), nil
	}

	wbrc.DialContext = dialer.DialContext

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

func fileName(link, contentType string) string {
	now := time.Now().Format("2006-01-02-150405")
	ext := "html"
	if exts, _ := mime.ExtensionsByType(contentType); len(exts) > 0 {
		ext = exts[0]
	}

	u, err := url.ParseRequestURI(link)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		return now + ext
	}

	domain := strings.ReplaceAll(u.Hostname(), ".", "-")
	if u.Path == "" || u.Path == "/" {
		return fmt.Sprintf("%s-%s%s", now, domain, ext)
	}

	baseName := path.Base(u.Path)
	if parts := strings.Split(baseName, "-"); len(parts) > 4 {
		baseName = strings.Join(parts[:4], "-")
	}

	return fmt.Sprintf("%s-%s-%s%s", now, domain, baseName, ext)
}
