package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"boscoin.io/sebak/lib/node"
	"github.com/BurntSushi/toml"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/spf13/cobra"
)

type FlagEnv []string

func (f *FlagEnv) Type() string {
	return "env"
}

func (f *FlagEnv) String() string {
	return ""
}

func (f *FlagEnv) Set(v string) error {
	parsed := strings.SplitN(v, "=", 2)
	if len(parsed) != 2 {
		return errors.New("invalid env")
	}

	*f = append(*f, v)

	return nil
}

type FlagVolume map[string]string

func (f *FlagVolume) Type() string {
	return "volume"
}

func (f *FlagVolume) String() string {
	return ""
}

func (f *FlagVolume) Set(v string) error {
	parsed := strings.SplitN(v, ":", 2)
	if len(parsed) != 2 {
		return errors.New("invalid volume")
	}

	n := map[string]string(*f)
	n[parsed[0]] = parsed[1]

	*f = n

	return nil
}

func PrintFlagsError(cmd *cobra.Command, flagName string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid '%s'; %v\n\n", flagName, err)
	}

	cmd.Help()

	os.Exit(1)
}

func PrintError(cmd *cobra.Command, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
	}

	cmd.Help()

	os.Exit(1)
}

type ListFlags []string

func (i *ListFlags) Type() string {
	return "list"
}

func (i *ListFlags) String() string {
	return strings.Join([]string(*i), " ")
}

func (i *ListFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type Volume struct {
	Source string
	Target string
}

func (v *Volume) UnmarshalText(b []byte) error {
	a := strings.SplitN(string(b), ":", 2)
	if len(a) != 2 {
		return fmt.Errorf("invalid volume: '%v'", string(b))
	}

	v.Source = a[0]
	v.Target = a[1]

	return nil
}

type DockerHost struct {
	Host   string   `toml:"host"`
	Cert   string   `toml:"cert"`
	Volume []Volume `toml:"volume"`
	Env    []string `toml:"env"`

	client *client.Client
	IP     string
	Nodes  []*node.LocalNode
}

func NewDockerHostFromURI(uri string) (dh *DockerHost, err error) {
	return
}

func (dh *DockerHost) CheckClient() (err error) {
	if dh.client != nil {
		return nil
	}

	var u *url.URL
	if u, err = url.Parse(dh.Host); err != nil {
		return
	}

	if len(dh.Cert) < 1 {
		err = fmt.Errorf("`cert` is missing")
		return
	}
	if strings.HasPrefix(dh.Cert, "~") {
		u, _ := user.Current()
		if len(dh.Cert) < 3 {
			dh.Cert = u.HomeDir
		} else {
			dh.Cert = filepath.Join(u.HomeDir, dh.Cert[2:])
		}
	}

	u.RawQuery = ""
	dh.Host = u.String()

	options := tlsconfig.Options{
		CAFile:             filepath.Join(dh.Cert, "ca.pem"),
		CertFile:           filepath.Join(dh.Cert, "cert.pem"),
		KeyFile:            filepath.Join(dh.Cert, "key.pem"),
		InsecureSkipVerify: os.Getenv("DOCKER_TLS_VERIFY") == "",
	}

	var tlsc *tls.Config
	if tlsc, err = tlsconfig.Client(options); err != nil {
		return err
	}

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsc,
		},
	}

	var cl *client.Client
	if cl, err = client.NewClient(dh.Host, "", c, nil); err != nil {
		return err
	}

	ctx := context.Background()
	_, err = cl.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}

	dh.client = cl

	return nil
}

func (dh *DockerHost) Client() *client.Client {
	return dh.client
}

func Ticker() chan bool {
	ch := make(chan bool)
	go func() {
		ticker := time.NewTicker(time.Second)
		go func() {
			for _ = range ticker.C {
				fmt.Fprint(os.Stderr, ".")
			}
		}()

		for {
			select {
			case <-ch:
				ticker.Stop()
				fmt.Fprint(os.Stderr, ".\n")
				break
			}
		}
	}()

	return ch
}

type Config struct {
	Genesis     string                `toml:"genesis"`
	DockerPath  string                `toml:"docker-path"`
	Hosts       map[string]DockerHost `toml:"hosts"`
	DockerHosts []*DockerHost
	dockerHosts map[string]*DockerHost
}

func (c *Config) GetDockerHost(host string) (dh *DockerHost, found bool) {
	dh, found = c.dockerHosts[host]
	return
}

func parseConfig(f string) (conf *Config, err error) {
	var i *os.File
	if i, err = os.Open(f); err != nil {
		return
	}

	var b []byte
	if b, err = ioutil.ReadAll(i); err != nil {
		return
	}

	if _, err = toml.Decode(string(b), &conf); err != nil {
		return
	}

	m := map[string]*DockerHost{}
	var keys []string
	for _, h := range conf.Hosts {
		dh := &DockerHost{
			Host:   h.Host,
			Cert:   h.Cert,
			Volume: h.Volume,
			Env:    h.Env,
		}
		if err = dh.CheckClient(); err != nil {
			return
		}

		m[dh.Host] = dh
		keys = append(keys, dh.Host)
	}

	for _, k := range keys {
		conf.DockerHosts = append(conf.DockerHosts, m[k])
	}

	conf.dockerHosts = m

	return
}

func HTTPGet(u string) (body []byte, err error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	var resp *http.Response
	if resp, err = client.Get(u); err != nil {
		return
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to get", "status", resp.StatusCode)
		return
	}

	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}

	return
}

func GetContainerName(s []string) string {
	if len(s) < 1 {
		return ""
	}

	return s[0][1:]
}
