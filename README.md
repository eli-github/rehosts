# Rehosts

forked from ([bitrate16/rehosts](https://github.com/bitrate16/rehosts))

CoreDNS hosts-like records plugin with support of regular expressions matching. Based on [hosts plugin](https://github.com/coredns/coredns/tree/master/plugin/hosts) functionality and extended with regexp.

# Prep 'plugin.cfg'

Run the script in CoreDNS main directory

```bash
 bash <(curl -s curl https://raw.githubusercontent.com/eli-github/rehosts/master/add_reauto_to_plugins.sh)
```

Occasionally the following command is required to be run, I'm not sure why at the moment

```google
 go get github.com/eli-github/rehosts
```

# Manual Install

Add this plugin to (`plugin.cfg` in coredns source)[https://github.com/coredns/coredns/blob/master/plugin.cfg]:
```

...
minimal:minimal
template:template
transfer:transfer

# Somewhere here to have regex priority voer stock hosts
rehosts:github.com/bitrate16/rehosts

rehosts:rehosts
hosts:hosts
route53:route53
...

```

Run to build coredns

```bash

make

```

And enjoy sample config:

```caddy

.:53 {
    reload 2s

    rehosts ./rehosts {
        ttl 3600
        reload 60s
        fallthrough
    }

    forward . 1.1.1.1

    log
    errors
    debug
}

```

# File syntax

File syntax is similar to `/etc/hosts` with additons:
```conf

# This is your hosts:
127.0.0.1 google.com


# This is your hosts on drugs:

# Regular, fully backwards-compatible with hosts
127.0.0.1 google.com
127.0.0.1 google.uk google.eu

# Wildcard or something. idk, I use cash
127.0.0.2 *.google.com
127.0.0.2 *.bing.com *.bing.cn
127.0.0.2 *.yeet.*

# regex, silly
127.0.0.3 @ ([a-n]+\.)?g?oogle\.(com|eu)

# unicode too
127.0.0.4 бебро.ед


```

# Behavior

Rules are evaluated sequently and in a strict order. This means if non-regex domain is placed above regex, it has priority over regex:
```conf

# In this example
127.0.0.1 foo.bar.baz
1.0.0.127   *.bar.baz

# Request for "NAME=foo.bar.baz" will return "127.0.0.1"

```

**NOTE:** Because if sequential evaulation, large amount of rules may cause performance drop.

# Blocklist example

```conf
127.0.0.1 analytics.google.com *.analytics.google.com
127.0.0.1 analytics.yandex.ru *.analytics.yandex.ru
127.0.0.1 *analytics.meta.com

127.0.0.1 @ .*telemetry.*
127.0.0.1 @ .*analytics.*
127.0.0.1 @ .*metrics.*
127.0.0.1 @ .*heartbeat.*

# e.t.c...
```
