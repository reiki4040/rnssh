package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"

	"github.com/reiki4040/peco"
)

func LoadSshConfigChoosableList() ([]peco.Choosable, error) {
	configs, err := ParseSshConfig()
	if err != nil {
		return nil, err
	}

	cList := make([]peco.Choosable, 0, len(configs))
	for _, c := range configs {
		w := new(tabwriter.Writer)
		var b bytes.Buffer
		w.Init(&b, 14, 0, 4, ' ', 0)
		fmt.Fprintf(w, "%s\t%s", c.Host, c.HostName)
		w.Flush()

		choice := &peco.Choice{
			C: b.String(),
			V: c.Host,
		}

		cList = append(cList, choice)
	}

	return cList, nil
}

type SshConfig struct {
	Host     string
	HostName string
}

func ParseSshConfig() ([]SshConfig, error) {
	user, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("can not resolved home dir: %s", err.Error())
	}
	path := filepath.Join(user.HomeDir, ".ssh", "config")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("not exists: %s", path)
	}

	fp, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	s := bufio.NewScanner(fp)
	hostRe := regexp.MustCompile(`Host\s+([^ #]+)`)
	hostnameRe := regexp.MustCompile(`HostName\s+([^ #]+)`)

	inHost := false
	configs := make([]SshConfig, 0)
	var host string
	for s.Scan() {
		line := s.Text()
		if !inHost {
			result := hostRe.FindAllStringSubmatch(line, -1)
			if len(result) == 0 {
				continue
			}

			host = result[0][1]
			if i := strings.Index(host, "*"); i != -1 {
				continue
			}

			inHost = true
		} else {
			result := hostnameRe.FindAllStringSubmatch(line, -1)
			if len(result) == 0 {
				continue
			}

			hostname := result[0][1]
			if i := strings.Index(hostname, "*"); i != -1 {
				continue
			}

			c := SshConfig{
				Host:     host,
				HostName: hostname,
			}

			configs = append(configs, c)
			inHost = false
		}
	}

	return configs, nil
}
