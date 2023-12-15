package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"

	"github.com/powerpuffpenguin/sf/config"
	"github.com/powerpuffpenguin/sf/forwarding"
	ver "github.com/powerpuffpenguin/sf/version"
)

func main() {
	u, err := url.Parse(`unix+tls://cf.socket/pp`)
	if err != nil {
		log.Fatalln(err)
	} else {
		fmt.Println(u)
		fmt.Println(u.Scheme)
		fmt.Println(u.Host)
		fmt.Println(u.Path)
		fmt.Println(u.User == nil)
		fmt.Println(u.Hostname())
		return
	}

	var (
		conf          string
		version, help bool
	)
	flag.StringVar(&conf, "conf", "", "Load config file path")
	flag.BoolVar(&version, "version", false, "Show version")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()
	if version {
		fmt.Printf(`sf-%s
%s/%s, %s, %s, %s
`,
			ver.Version,
			runtime.GOOS, runtime.GOARCH,
			runtime.Version(),
			ver.Date, ver.Commit,
		)
		return
	} else if help {
		flag.PrintDefaults()
		return
	} else if conf == `` {
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.SetFlags(log.Lshortfile | log.LstdFlags)
	var c config.Config
	e := c.Load(conf)
	if e != nil {
		log.Fatalln(e)
	}
	app, e := forwarding.NewApplication(&c)
	if e != nil {
		log.Fatalln(e)
		return
	}
	app.Serve()
}
