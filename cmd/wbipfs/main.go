package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sync"

	"github.com/wabarc/wbipfs"
)

func main() {
	host := flag.String("host", "127.0.0.1", "IPFS host")
	port := flag.Uint("port", 5001, "IPFS port")
	mode := flag.String("mode", "pinner", "IPFS running mode supports daemon pinner, daemon requires to run an ipfs standalone daemon")
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

	var wg sync.WaitGroup
	for _, arg := range args {
		wg.Add(1)
		go func(link string) {
			defer wg.Done()
			input, err := url.Parse(link)
			if err != nil {
				fmt.Println(link, "=>", fmt.Sprintf("%v", err))
				return
			}

			dst, err := wbrc.Wayback(context.Background(), input)
			if err != nil {
				fmt.Println(link, "=>", fmt.Sprintf("%v", err))
				return
			}
			fmt.Println(link, "=>", dst)
		}(arg)
	}
	wg.Wait()
}
