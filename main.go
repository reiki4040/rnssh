package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"
	flag "github.com/dotcloud/docker/pkg/mflag"

	"github.com/reiki4040/peco"
	myec2 "github.com/reiki4040/rnssh/internal/ec2"
)

const (
	Usage = `rnssh

usage:

  rnssh [-f] [-p] [-l SSH_USER] [-s] query strings ...
  rnssh -h
  rnssh -v
  rnssh -d

options:
  -f: reload ec2 instances infomaion. connect to AWS.
      you have to specify this option after you modified instances.
	  (launch, start, stop, modify name, etc...)

  -P: use Public IP address. this is default ssh host type.
      if you have set RNSSH_HOST_TYPE variable and value is private/name,
      you can ssh with Public IP temporarily with this option.

  -p: use Private IP address. for VPN/Direct connect.
      you can ssh with Private IP always with RNSSH_HOST_TYPE environment variable.

  -n: use Name tag.
      this option for ssh config that Host named by ec2 Name tag.

      # example ssh config
      Host EC2-Name-tag
        HostName X.X.X.X (or domain)
        User ec2-user
        IdentityFile path/to/key

      you can ssh with Name tag always with RNSSH_HOST_TYPE environment variable.

  -r: target region. if you set 'AWS_REGION' environment variable, use it.
      if you specify both, use -r options.

  -s: show ssh command string that would be run. (debug)

options for ssh:
  -l: ssh user.
  -i: identity file path.

options for help:
  -h: show this usage.
  -v: show version.

args:

  query string...: filtering ec2 instances list.

Environment variables:

  RNSSH_HOST_TYPE: this variable can specify default ssh host types.

    public  : use Public IP (default)
    private : use Private IP
    name    : use Name tag

Caution
*caution to option and arguments order*
  the options must be before arguments.

  - working
  rnssh -s query string

  - not working
  rnssh query string -s
`

	ENV_RNSSH_HOST_TYPE = "RNSSH_HOST_TYPE"

	HOST_TYPE_PUBLIC_IP  = "public"
	HOST_TYPE_PRIVATE_IP = "private"
	HOST_TYPE_NAME_TAG   = "name"
)

var (
	version   string
	hash      string
	builddate string

	show_version bool
	show_usage   bool

	force_reload bool
	regionName   string

	optPrivateIP bool
	optPublicIP  bool
	optNameTag   bool

	showCommand bool

	optSshUser      string
	optIdentityFile string
)

func init() {
	flag.BoolVar(&show_version, []string{"v", "-version"}, false, "show version.")
	flag.BoolVar(&show_usage, []string{"h", "-help"}, false, "show this usage.")

	flag.BoolVar(&force_reload, []string{"f", "-force"}, false, "reload ec2 (force connect to AWS)")
	flag.BoolVar(&optPublicIP, []string{"P", "-public-ip"}, false, "ssh with EC2 Public IP")
	flag.BoolVar(&optPrivateIP, []string{"p", "-private-ip"}, false, "ssh with EC2 Private IP")
	flag.BoolVar(&optNameTag, []string{"n", "-name-tag"}, false, "ssh with EC2 Name tag")
	flag.BoolVar(&showCommand, []string{"s", "-show-command"}, false, "show ssh command that will do (debug)")

	flag.StringVar(&regionName, []string{"r", "-region"}, "", "specify region")
	flag.StringVar(&optSshUser, []string{"l", "-user"}, "", "specify ssh user")
	flag.StringVar(&optIdentityFile, []string{"i", "-identity-file"}, "", "specify ssh identity file")

	flag.Parse()
}

func showVersion() {
	fmt.Printf("%s (%s) built:%s\n", version, hash, builddate)
}

func usage() {
	fmt.Printf("%s\n", Usage)
}

func duplicateHostTypeOption(public, private, name bool) bool {
	return public && private || private && name || public && name
}

func main() {
	if show_usage {
		usage()
		os.Exit(0)
	}

	if show_version {
		showVersion()
		os.Exit(0)
	}

	if duplicateHostTypeOption(optPublicIP, optPrivateIP, optNameTag) {
		fmt.Printf("duplicate specify option -P/-p/-n. please spcify only one.\n")
		os.Exit(1)
	}

	if optIdentityFile != "" {
		if _, err := os.Stat(optIdentityFile); os.IsNotExist(err) {
			fmt.Printf("Identity file not exists: %s\n", optIdentityFile)
			os.Exit(1)
		}
	}

	region := myec2.GetRegion(regionName)
	if region == "" {
		fmt.Println("region is empty. please specify by region option (-r) or AWS_REGION envirnment variable.")
		os.Exit(1)
	}

	err := myec2.CreateRnzooDir()
	if err != nil {
		fmt.Printf("can not create rnzoo dir: %s\n", err.Error())
		os.Exit(1)
	}

	instances, err := myec2.GetEC2Array(force_reload, region)
	if err != nil {
		fmt.Printf("failed ec2 list: %s\n", err.Error())
		os.Exit(1)
	}

	if len(instances) == 0 {
		fmt.Printf("there is no instance. not running\n", region)
		os.Exit(1)
	}

	// support user@host format
	sshUser, hostname, err := getSshUserAndHostname(strings.Join(flag.Args(), " "))
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	// show ec2 instances and choose intactive
	targetHost, err := chooseEC2Instance(instances, hostname)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	sshHost, err := getSshHost(optPublicIP, optPrivateIP, optNameTag, targetHost)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	sshArgs := genSshArgs(optSshUser, optIdentityFile, sshUser, sshHost)

	if showCommand {
		fmt.Printf("%s %s\n", "ssh", strings.Join(sshArgs, " "))
		os.Exit(0)
	}

	cmd := exec.Command("ssh", sshArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func getSshUserAndHostname(sshTarget string) (string, string, error) {
	// support user@host format
	idx := strings.Index(sshTarget, "@")

	// not include(-1) or first char(0)
	if idx > 0 {
		splited := strings.SplitN(sshTarget, "@", 2)
		return splited[0], splited[1], nil
	}

	return "", sshTarget, nil
}

func chooseEC2Instance(instances []*ec2.Instance, defaultQuery string) (*ChoosableEC2, error) {
	choices := convertChoosableList(instances)
	if len(choices) == 0 {
		err := fmt.Errorf("there is no running instance.")
		return nil, err
	}
	pecoOpt := &peco.PecoOptions{
		OptPrompt: "which do you choose the instance? >",
		OptQuery:  defaultQuery,
	}

	result, err := peco.PecolibWithOptions(choices, pecoOpt)
	if err != nil || len(result) == 0 {
		err := fmt.Errorf("no select target.")
		return nil, err
	}

	var targetHost *ChoosableEC2
	for _, r := range result {
		if ec2host, ok := r.(*ChoosableEC2); ok {
			targetHost = ec2host
		} else {
			err := fmt.Errorf("this is bug. type is not ChoosableEC2: %v", r)
			return nil, err
		}
	}

	return targetHost, nil
}

func convertChoosableList(instances []*ec2.Instance) []peco.Choosable {
	choices := make([]peco.Choosable, 0, len(instances))
	for _, i := range instances {
		c := convertChoosable(i)
		if c != nil {
			choices = append(choices, c)
		}
	}

	return choices
}

func convertChoosable(i *ec2.Instance) *ChoosableEC2 {
	if i.State.Name != nil {
		s := i.State.Name
		if *s != "running" {
			return nil
		}
	} else {
		return nil
	}

	var nameTag string
	for _, tag := range i.Tags {
		if convertNilString(tag.Key) == "Name" {
			nameTag = convertNilString(tag.Value)
			break
		}
	}

	ins := *i
	ec2host := &ChoosableEC2{
		InstanceId:       convertNilString(ins.InstanceId),
		Name:             nameTag,
		IPAddress:        convertNilString(ins.PublicIpAddress),
		PrivateIPAddress: convertNilString(ins.PrivateIpAddress),
	}

	return ec2host
}

func convertNilString(s *string) string {
	if s == nil {
		return ""
	} else {
		return *s
	}
}

type ChoosableEC2 struct {
	InstanceId       string
	Name             string
	IPAddress        string
	PrivateIPAddress string
}

func (e *ChoosableEC2) Choice() string {
	ipAddr := e.IPAddress
	if ipAddr == "" {
		ipAddr = "NO PUBLIC IP"
	}

	return fmt.Sprintf("%s\t[%s]\t[%s]\t[%s]", e.InstanceId, e.Name, ipAddr, e.PrivateIPAddress)
}

func getSshHost(publicIP, privateIP, nameTag bool, targetHost *ChoosableEC2) (string, error) {

	sshHost := ""
	// default host type.
	hostType := os.Getenv(ENV_RNSSH_HOST_TYPE)

	// overwrite by option
	if publicIP {
		hostType = HOST_TYPE_PUBLIC_IP
	}

	if privateIP {
		hostType = HOST_TYPE_PRIVATE_IP
	}

	if nameTag {
		hostType = HOST_TYPE_NAME_TAG
	}

	switch hostType {
	case "":
		// default public ip
		fallthrough
	case HOST_TYPE_PUBLIC_IP:
		return targetHost.IPAddress, nil
	case HOST_TYPE_PRIVATE_IP:
		return targetHost.PrivateIPAddress, nil
	case HOST_TYPE_NAME_TAG:
		return targetHost.Name, nil
	default:
		err := fmt.Errorf("unknown ssh type: %s\n", sshHost)
		return "", err
	}
}

func genSshArgs(optSshUser, optIdentityFile, sshUser, sshHost string) []string {
	args := make([]string, 0)
	if optSshUser != "" {
		args = append(args, "-l "+optSshUser)
	}

	if optIdentityFile != "" {
		args = append(args, "-i "+optIdentityFile)
	}

	if sshUser != "" {
		sshHost = sshUser + "@" + sshHost
	}

	args = append(args, sshHost)

	return args
}
