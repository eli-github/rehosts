package rehosts

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/net/idna"
)

type Matcher interface {
	Match(*string) bool
}

// Single hosts file record that maps regex matcher to IP (v4 or v6)
type RehostsFileRecord struct {
	Match  func(str string) bool
	AddrV4 []net.IP
	AddrV6 []net.IP
}

type options struct {
	// Auto reload period
	reload time.Duration

	// TTL of DNS record
	ttl uint32
}

func newOptions() *options {
	return &options{
		ttl:    3600,
		reload: 5 * time.Second,
	}
}

type RehostsFile struct {
	// DNS Regex records
	records []*RehostsFileRecord

	// List pf authoritative origins
	Origins []string

	// File attrubutes for relaod check
	mtime time.Time
	fsize int64

	// Update lock
	sync.RWMutex

	// Path to file
	path string

	// Options from Caddyfile
	options *options
}

func (r *RehostsFile) readRehosts() {
	file, err := os.Open(r.path)
	if err != nil {
		return
	}
	defer file.Close()

	// Check if file has changed
	stat, err := file.Stat()
	if err != nil {
		return
	}
	r.RLock()
	fsize := r.fsize
	mtime := r.mtime
	r.RUnlock()

	if mtime.Equal(stat.ModTime()) && fsize == stat.Size() {
		return
	}

	newRecords := r.parse(file)
	log.Debugf("Parsed rehosts file into %d entries", len(newRecords))

	r.Lock()

	r.records = newRecords
	r.mtime = stat.ModTime()
	r.fsize = stat.Size()

	r.Unlock()
}

func parseIP(addr string) net.IP {
	addr = strings.TrimSpace(addr)

	// discard IPv6 zone (lol?)
	if pos := strings.Index(addr, "%"); pos >= 0 {
		addr = addr[0:pos]
	}

	return net.ParseIP(addr)
}

func verifyWildcard(s string) bool {
	for _, c := range s {
		if unicode.IsLetter(c) {
			continue
		}
		if unicode.IsDigit(c) {
			continue
		}
		if (c == '*') || (c == '.') || (c == '-') || (c == '_') {
			continue
		}
		return false
	}
	return true
}

// Parse reads the hostsfile and populates the byName and addr maps.
func (h *RehostsFile) parse(r io.Reader) []*RehostsFileRecord {
	records := make([]*RehostsFileRecord, 0)
	wildcardReplacer := strings.NewReplacer(".", "\\.", "*", ".*")

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()

		// Remove all comments
		if commentPos := bytes.Index(line, []byte{'#'}); commentPos >= 0 {
			line = line[0:commentPos]
		}
		line = bytes.TrimSpace(line)

		if len(line) == 0 {
			continue
		}

		// Regex mode
		if atPos := bytes.Index(line, []byte{'@'}); atPos >= 0 {
			// Try parse IP
			ipStr := string(line[0:atPos])
			ip := parseIP(ipStr)
			if ip == nil {
				log.Warningf("Invalid ip %q", ipStr)
				continue
			}

			// Try parse regexp
			regexpStr := string(bytes.TrimSpace(line[atPos+1:]))
			regexp, err := regexp.Compile(regexpStr)
			if err != nil {
				log.Warningf("Invalid regexp %q: %v", regexp, err)
				continue
			}
			// TODO: Check for authoritative zones?

			// Combine together
			var record RehostsFileRecord
			record.Match = func(str string) bool {
				return regexp.MatchString(str)
			}
			if ip.To4() != nil {
				record.AddrV4 = append(record.AddrV4, ip)
			} else {
				record.AddrV6 = append(record.AddrV6, ip)
			}

			records = append(records, &record)
		} else {
			fields := bytes.Fields(line)

			// Try parse IP
			ipStr := string(fields[0])
			ip := parseIP(ipStr)
			if ip == nil {
				log.Warningf("Invalid ip %q", ipStr)
				continue
			}

			for fieldIndex := 1; fieldIndex < len(fields); fieldIndex++ {
				fieldStr := string(fields[fieldIndex])

				// Single record per each domain in line
				var record RehostsFileRecord
				if ip.To4() != nil {
					record.AddrV4 = append(record.AddrV4, ip)
				} else {
					record.AddrV6 = append(record.AddrV6, ip)
				}

				// Check if addr is some kind of wildcard
				if wcPos := strings.Index(fieldStr, "*"); wcPos >= 0 {
					// Normalize
					if !verifyWildcard(fieldStr) {
						log.Warningf("Invalid wildcard %q", fieldStr)
						continue
					}
					regexpStr := wildcardReplacer.Replace(fieldStr)
					regexpStr = strings.ToLower(regexpStr)

					// Try parse regexp
					regexp, err := regexp.Compile(regexpStr)
					if err != nil {
						log.Warningf("Invalid regexp %q: %v", regexp, err)
						continue
					}
					// TODO: Check for authoritative zones?

					record.Match = func(str string) bool {
						return regexp.MatchString(str)
					}
				} else {
					// Normalize
					hostName := strings.ToLower(fieldStr)

					record.Match = func(str string) bool {
						return hostName == str
					}
				}

				records = append(records, &record)
			}
		}
	}

	return records
}

func DeFQDNnIDNA(host string) (string, error) {
	if !(len(host) > 0 && host[len(host)-1] == '.') {
		return "", errors.New("not FQDN")
	}
	host = host[:len(host)-1]
	host = strings.ToLower(host)

	unicodeHost, err := idna.ToUnicode(host)
	if err != nil {
		return "", err
	}

	return unicodeHost, nil
}

// Lookup host IPv4 records
func (r *RehostsFile) LookupStaticHostV4(host string) []net.IP {
	r.RLock()
	defer r.RUnlock()

	if r.records == nil {
		return nil
	}

	for _, record := range r.records {
		unicodeHost, err := DeFQDNnIDNA(host)
		if err != nil {
			log.Debugf("Invalid IDNA %q: %v", host, err)
			return nil
		}

		if record.Match(unicodeHost) && len(record.AddrV4) != 0 {
			addr4Copy := make([]net.IP, len(record.AddrV4))
			copy(addr4Copy, record.AddrV4)
			return addr4Copy
		}
	}

	return nil
}

// Lookup host IPv6 records
func (r *RehostsFile) LookupStaticHostV6(host string) []net.IP {
	r.RLock()
	defer r.RUnlock()

	if r.records == nil {
		return nil
	}

	for _, record := range r.records {
		unicodeHost, err := DeFQDNnIDNA(host)
		if err != nil {
			log.Debugf("Invalid IDNA %q: %v", host, err)
			return nil
		}

		if record.Match(unicodeHost) && len(record.AddrV6) != 0 {
			addr6Copy := make([]net.IP, len(record.AddrV6))
			copy(addr6Copy, record.AddrV6)
			return addr6Copy
		}
	}

	return nil
}
