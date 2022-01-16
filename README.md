# A Golang and Command-Line Interface to IPFS

**This package has been migrated to [rivet](https://github.com/wabarc/rivet).**

This package is a command-line tool named `wbipfs` saving webpage to IPFS. It also supports imports as a Golang package for a programmatic. Please report all bugs and issues on [Github](https://github.com/wabarc/wbipfs/issues).

## Installation

```sh
$ go get -u -v github.com/wabarc/wbipfs/cmd/wbipfs
```

## Usage

#### Command-line

```sh
$ wbipfs --help
version: 0.1.0
date: 2020/09/19

Usage of wbipfs:
  -host string
        IPFS host (default "127.0.0.1")
  -mode string
        IPFS running mode supports daemon and pinner, daemon requires to run an ipfs standalone daemon. (default "pinner")
  -port uint
        IPFS port (default 5001)
  -tor
        Wayback webpage with Tor proxy, it requires tor executable
```

```sh
$ wbipfs https://www.google.com https://www.bbc.com

Output:
version: 0.1.0
date: 2020/09/19

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

## FAQ

#### Optional to disable JavaScript for some URI?

If you prefer to disable JavaScript on saving webpage, you could add environmental variables `DISABLEJS_URIS`
and set the values with the following formats:

```sh
export DISABLEJS_URIS=wikipedia.org|eff.org/tags
```

It will disable JavaScript for domain of the `wikipedia.org` and path of the `eff.org/tags` if matching it.

## Credits

- [IPFS](https://ipfs.io/)
- [go-shiori/obelisk](https://github.com/go-shiori/obelisk)

## License

This software is released under the terms of the GNU General Public License v3.0, see the [LICENSE](https://github.com/wabarc/wbipfs/blob/main/LICENSE) file for details.

