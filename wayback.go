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
	"github.com/wabarc/logger"
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
func (wbrc *Archiver) Wayback(ctx context.Context, input *url.URL) (dst string, err error) {
	// Write content to tmp file
	dir, err := ioutil.TempDir(os.TempDir(), "wbipfs-")
	if err != nil {
		logger.Error("[wbipfs] create temporary directory failed: %v", err)
		return dst, fmt.Errorf("Create temp directory failed: %v", err)
	}
	defer os.RemoveAll(dir)

	if wbrc.UseTor {
		if err, tor := wbrc.dial(); err != nil {
			logger.Error("[wbipfs] dial tor failed: %v", err)
			return dst, fmt.Errorf("Dial tor failed: %w", err)
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
		DialContext: wbrc.context,
		DisableJS:   disableJS(uri),

		SkipResourceURLError: true,
	}
	arc.Validate()

	dst = "Archive failed."
	content, contentType, err := arc.Archive(context.Background(), req)
	if err != nil {
		log.Printf("Archive failed: %s", err)
		return dst, err
	}

	filepath := filepath.Join(dir, fileName(uri, contentType))
	if err := ioutil.WriteFile(filepath, content, 0666); err != nil {
		logger.Error("Write failed, path: %s, err: %s", filepath, err)
		return dst, err
	}

	switch wbrc.IPFSMode {
	case "daemon":
		// Valid IPFS daemon connection
		if wbrc.IPFSHost == "" || wbrc.IPFSPort == 0 || wbrc.IPFSPort > 65535 {
			logger.Error("IPFS hostname or port is invalid, host: %s, port: %d", wbrc.IPFSHost, wbrc.IPFSPort)
			return dst, fmt.Errorf("IPFS hostname or port is invalid")
		}
		worker := NewDaemon(wbrc.IPFSHost, wbrc.IPFSPort)
		cid, err := worker.Transfer(filepath)
		if err != nil {
			logger.Error("Transfer failed, path: %s, err: %s", filepath, err)
			break
		}
		dst = fmt.Sprintf("https://ipfs.io/ipfs/%s#%s", cid, uri)
	case "pinner":
		if cid, err := Pinner(filepath); err != nil {
			logger.Error("Pin failed, path: %s, err: %s", filepath, err)
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

func (wbrc *Archiver) dial() (error, *tor.Tor) {
	// Lookup tor executable file
	if _, err := exec.LookPath("tor"); err != nil {
		return fmt.Errorf("%w", err), nil
	}

	// Start tor with default config
	t, err := tor.Start(context.TODO(), nil)
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
