package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/powerpuffpenguin/streamf/config"
	"github.com/powerpuffpenguin/streamf/forwarding"
	ver "github.com/powerpuffpenguin/streamf/version"
)

func main() {
	var (
		conf          string
		version, help bool
		logLevel      string
	)
	flag.StringVar(&conf, "conf", "", "Load config file path")
	flag.StringVar(&logLevel, "log", "", "Log level [debug info warn error]")
	flag.BoolVar(&version, "version", false, "Show version")
	flag.BoolVar(&help, "help", false, "Show help")
	flag.Parse()
	if version {
		fmt.Printf(`streamf-%s
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
	if logLevel != `` {
		c.Logger.Level = logLevel
	}
	app, e := forwarding.NewApplication(&c)
	if e != nil {
		return
	}
	app.Serve()
}
