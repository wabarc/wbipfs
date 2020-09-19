package wbipfs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	shell "github.com/ipfs/go-ipfs-api"

	"github.com/wabarc/ipfs-pinner/pkg/infura"
	"github.com/wabarc/ipfs-pinner/pkg/pinata"
)

func Publish(s string) (string, error) {
	return "", fmt.Errorf("%s", "embed mode removed.")
}

// Publish to IPFS use local daemon
type Daemon struct {
	Host string
	Port uint
}

func NewDaemon(host string, port uint) *Daemon {
	dm := &Daemon{
		Host: "localhost",
		Port: 5001,
	}
	if len(host) > 0 {
		dm.Host = host
	}
	if port < 65536 || port > 0 {
		dm.Port = port
	}

	return dm
}

func (dm *Daemon) Transfer(source string) (string, error) {
	if len(dm.Host) == 0 || dm.Port > 65535 || dm.Port < 1 {
		return "", fmt.Errorf("IPFS host or port is not valid")
	}

	content, err := ioutil.ReadFile(source)
	if err != nil {
		return "", fmt.Errorf("Read file failed: %w", err)
	}

	sh := shell.NewShell(fmt.Sprintf("%s:%d", dm.Host, dm.Port))
	cid, err := sh.Add(strings.NewReader(string(content)))
	if err != nil {
		return "", fmt.Errorf("Add file to IPFS failed: %w", err)
	}

	return cid, nil
}

// Pinner is the handle of put the file to IPFS by pinning service.
// It supports two slots which infura and pinata, you can apply a specific slot via WAYBACK_SLOT env.
// If you use the pinata slot, it requires WAYBACK_APIKEY and WAYBACK_SECRET environment variable.
func Pinner(source string) (string, error) {
	var err error
	var cid string
	var slot string = os.Getenv("WAYBACK_SLOT")

	switch slot {
	default:
		cid, err = infura.PinFile(source)
	case "infura":
		cid, err = infura.PinFile(source)
	case "pinata":
		apikey := os.Getenv("WAYBACK_APIKEY")
		secret := os.Getenv("WAYBACK_SECRET")
		if apikey == "" || secret == "" {
			log.Println("Please set WAYBACK_APIKEY or WAYBACK_SECRET env.")
			err = fmt.Errorf("Missing WAYBACK_APIKEY or WAYBACK_SECRET env.")
			break
		}
		pnt := &pinata.Pinata{Apikey: apikey, Secret: secret}
		cid, err = pnt.PinFile(source)
	}

	if err != nil {
		return "", err
	} else {
		return cid, nil
	}
}
