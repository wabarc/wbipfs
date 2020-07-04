# A Golang and Command-Line Interface to IPFS

This package is a command-line tool named `wbipfs` saving webpage to IPFS. It also supports imports as a Golang package for a programmatic. Please report all bugs and issues on [Github](https://github.com/wabarc/wbipfs/issues).

## Installation

```sh
$ go get -u -v github.com/wabarc/wbipfs/...
```

## Usage

#### Command-line

```sh
$ wbipfs --help
version: 0.0.1
date: unknown

Usage of wbipfs:
  -host string
        IPFS host (default "127.0.0.1")
  -mode string
        IPFS running mode supports daemon and embed, daemon requires to run an ipfs standalone daemon, embed now is experimental (default "daemon")
  -port uint
        IPFS port (default 5001)
  -tor
        Wayback webpage with Tor proxy, it requires tor executable
```

```sh
$ wbipfs https://www.google.com https://www.bbc.com

Output:
version: 0.0.1
date: unknown

https://www.google.com => https://ipfs.io/ipfs/QmSGvyuAGiwQHTeAzYEhfhhZbhvyCN6PX1kCq3vgwmkPmU
https://www.bbc.com => https://ipfs.io/ipfs/QmXvUs1ic7uPtfxn7iQHfbefzcrrmnSYP8YDE4BU6jEUab
```

#### Go package interfaces

```go
package main

import (
	"fmt"

	"github.com/wabarc/wbipfs"
)

func main() {
        wbrc := &wbipfs.Archiver{}
        links := []string{"https://www.google.com", "https://www.bbc.com"}
        urls, _ := wbrc.Wayback(args)
        for orig, dest := range urls {
                fmt.Println(orig, "=>", dest)
        }
}

// Output:
// https://www.google.com => https://ipfs.io/ipfs/QmSAQ2DYMfRaPgnoWnAgyBYhJXEV5G4dApeukf6yYbnqyE
// https://www.bbc.com => https://ipfs.io/ipfs/QmUXFPSJEXcXaZB9WthxbkmYWvUw1JCiNYGWDAAr7jd88p
```

## Credits

- [IPFS](https://ipfs.io/)
- [go-shiori/obelisk](https://github.com/go-shiori/obelisk)

## License

Permissive GPL 3.0 license, see the [LICENSE](https://github.com/wabarc/wbipfs/blob/master/LICENSE) file for details.

