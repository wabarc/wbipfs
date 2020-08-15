package wbipfs

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
	config "github.com/ipfs/go-ipfs-config"
	files "github.com/ipfs/go-ipfs-files"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	icore "github.com/ipfs/interface-go-ipfs-core"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/wabarc/ipfs-pinner/pkg/infura"
	"github.com/wabarc/ipfs-pinner/pkg/pinata"
)

func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		// return fmt.Errorf("error initializing plugins: %s", err)
		return nil
	}

	return nil
}

func createTempRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "wbrc-ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		// return "", fmt.Errorf("failed to init ephemeral node: %s", err)
		return repoPath, nil
	}

	return repoPath, nil
}

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}

// Spawns a node on the default repo location, if the repo exists
func spawnDefault(ctx context.Context) (icore.CoreAPI, error) {
	defaultPath, err := config.PathRoot()
	if err != nil {
		// shouldn't be possible
		return nil, err
	}

	if err := setupPlugins(defaultPath); err != nil {
		return nil, err

	}

	return createNode(ctx, defaultPath)
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawnEphemeral(ctx context.Context) (icore.CoreAPI, error) {
	if err := setupPlugins(""); err != nil {
		return nil, err
	}

	// Create a Temporary Repo
	repoPath, err := createTempRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// Spawning an ephemeral IPFS node
	return createNode(ctx, repoPath)
}

func connectToPeers(ctx context.Context, ipfs icore.CoreAPI, peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				// log.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	f, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func getPeers() []string {
	file, err := os.Open("../configs/peers")
	if err != nil {
		return []string{}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines
}

func Publish(source string) (string, error) {
	// Getting a IPFS node running
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spawn a node using a temporary path, creating a temporary repo for the run
	ipfs, err := spawnEphemeral(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to spawn ephemeral node: %s", err)
	}

	// Adding a file to IPFS
	someFile, err := getUnixfsNode(source)
	if err != nil {
		return "", fmt.Errorf("Could not get File: %s", err)
	}

	cidFile, err := ipfs.Unixfs().Add(ctx, someFile)
	if err != nil {
		return "", fmt.Errorf("Could not add File: %s", err)
	}

	// Getting the file you added back
	outputBasePath := os.TempDir() + "/wbipfs-out-"
	outputPathFile := outputBasePath + strings.Split(cidFile.String(), "/")[2]

	rootNodeFile, err := ipfs.Unixfs().Get(ctx, cidFile)
	if err != nil {
		return "", fmt.Errorf("Could not get file with CID: %s", err)
	}

	err = files.WriteTo(rootNodeFile, outputPathFile)
	if err != nil {
		return "", fmt.Errorf("Could not write out the fetched CID: %s", err)
	}
	defer os.RemoveAll(outputPathFile)

	// Getting a file from the IPFS Network
	go connectToPeers(ctx, ipfs, getPeers())

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	if resp, err := client.Get("https://ipfs.io" + cidFile.String()); err != nil {
		resp.Body.Close()
	}

	return cidFile.String(), nil
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
