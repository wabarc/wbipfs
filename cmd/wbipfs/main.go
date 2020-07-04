package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wabarc/wbipfs"
)

func main() {
	host := flag.String("host", "127.0.0.1", "IPFS host")
	port := flag.Uint("port", 5001, "IPFS port")
	mode := flag.String("mode", "daemon", "IPFS running mode supports daemon and embed, daemon requires to run an ipfs standalone daemon, embed now is experimental")
	tor := flag.Bool("tor", false, "Wayback webpage with Tor proxy, it requires tor executable")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		e := os.Args[0]
		fmt.Printf("  %s [-host=127.0.0.1] [-port=5001] [-mode=daemon] [-tor] url [url1] [url2] [urln]\n\n", e)
		fmt.Printf("example:\n  %s https://www.google.com https://www.bbc.com\n\n", e)
		os.Exit(1)
	}

	wbrc := &wbipfs.Archiver{
		IPFSHost: *host,
		IPFSPort: *port,
		IPFSMode: *mode,
		UseTor:   *tor,
	}

	links, _ := wbrc.Wayback(args)
	for orig, dest := range links {
		fmt.Println(orig, "=>", dest)
	}
}
