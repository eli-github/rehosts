package rehosts

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("rehosts")

func init() { plugin.Register("rehosts", setup) }

func periodicRehostsReload(r *Rehosts) chan struct{} {
	exitChan := make(chan struct{})

	if r.options.reload == 0 {
		return exitChan
	}

	go func() {
		tickerChan := time.NewTicker(r.options.reload)
		defer tickerChan.Stop()

		for {
			select {
			case <-exitChan:
				return
			case <-tickerChan.C:
				r.readRehosts()
			}
		}
	}()

	return exitChan
}

func setup(c *caddy.Controller) error {
	r, err := parseConfig(c)
	if err != nil {
		plugin.Error("rehosts", err)
	}

	closeChan := periodicRehostsReload(&r)

	c.OnStartup(func() error {
		r.readRehosts()
		return nil
	})

	c.OnShutdown(func() error {
		close(closeChan)
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		r.Next = next
		return r
	})

	return nil
}

func parseConfig(c *caddy.Controller) (Rehosts, error) {
	config := dnsserver.GetConfig(c)

	rh := Rehosts{
		RehostsFile: &RehostsFile{
			path:    "/etc/hosts",
			options: newOptions(),
		},
	}

	i := 0
	for c.Next() {
		if i > 0 {
			return rh, plugin.ErrOnce
		}
		i++

		args := c.RemainingArgs()

		if len(args) >= 1 {
			rh.path = args[0]
			args = args[1:]

			if !filepath.IsAbs(rh.path) && config.Root != "" {
				rh.path = filepath.Join(config.Root, rh.path)
			}
			s, err := os.Stat(rh.path)
			if err != nil {
				if os.IsNotExist(err) {
					log.Warningf("File does not exist: %s", rh.path)
				} else {
					return rh, c.Errf("unable to access hosts file '%s': %v", rh.path, err)
				}
			}
			if s != nil && s.IsDir() {
				log.Warningf("Hosts file %q is a directory", rh.path)
			}
		}

		rh.Origins = plugin.OriginsFromArgsOrServerBlock(args, c.ServerBlockKeys)

		for c.NextBlock() {
			switch c.Val() {
			case "fallthrough":
				rh.Fall.SetZonesFromArgs(c.RemainingArgs())
			case "ttl":
				remaining := c.RemainingArgs()
				if len(remaining) < 1 {
					return rh, c.Errf("ttl needs a time in second")
				}
				ttl, err := strconv.Atoi(remaining[0])
				if err != nil {
					return rh, c.Errf("ttl needs a number of second")
				}
				if ttl <= 0 || ttl > 65535 {
					return rh, c.Errf("ttl provided is invalid")
				}
				rh.options.ttl = uint32(ttl)
			case "reload":
				remaining := c.RemainingArgs()
				if len(remaining) != 1 {
					return rh, c.Errf("reload needs a duration (zero seconds to disable)")
				}
				reload, err := time.ParseDuration(remaining[0])
				if err != nil {
					return rh, c.Errf("invalid duration for reload '%s'", remaining[0])
				}
				if reload < 0 {
					return rh, c.Errf("invalid negative duration for reload '%s'", remaining[0])
				}
				rh.options.reload = reload
			default:
				return rh, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	return rh, nil
}
