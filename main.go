package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"

	flag "github.com/dotcloud/docker/pkg/mflag"

	"github.com/reiki4040/peco"
	myec2 "github.com/reiki4040/rnssh/internal/ec2"
	"github.com/reiki4040/rnssh/internal/rnssh"
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
  --port: ssh port.
  --strict-host-key-checking-no: suppress host key checking.
                                 using 'ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'

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

	optSshUser                 string
	optIdentityFile            string
	optPort                    int
	optStrictHostKeyCheckingNo bool
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
	flag.IntVar(&optPort, []string{"-port"}, 0, "specify ssh port")
	flag.BoolVar(&optStrictHostKeyCheckingNo, []string{"-strict-host-key-checking-no"}, false, "suppress host key checking. => 'ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'")

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
		// replace ~ -> home dir
		if i := strings.Index(optIdentityFile, "~"); i == 0 {
			user, err := user.Current()
			if err != nil {
				fmt.Printf("can not resolved home dir: %s\n", err.Error())
				os.Exit(1)
			}
			optIdentityFile = user.HomeDir + "/" + optIdentityFile[1:]
		}

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

	err := rnssh.CreateRnsshDir()
	if err != nil {
		fmt.Printf("can not create rnzoo dir: %s\n", err.Error())
		os.Exit(1)
	}

	// support user@host format
	sshUser, hostname, err := getSshUserAndHostname(strings.Join(flag.Args(), " "))
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	sshTargetType := getSshTargetType(optPublicIP, optPrivateIP, optNameTag)

	handler := myec2.DefaultEC2Handler()
	choosableList, err := handler.LoadTargetHost(sshTargetType, region, force_reload)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	if len(choosableList) == 0 {
		fmt.Printf("there is no instance. not running %s\n", region)
		os.Exit(1)
	}

	// show ec2 instances and choose intactive
	targetHost, err := chooseTargetHost(choosableList, hostname)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	sshHost := targetHost.GetSshTarget()
	sshArgs := genSshArgs(optSshUser, optIdentityFile, optPort, optStrictHostKeyCheckingNo, sshUser, sshHost)

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

func chooseTargetHost(choices []rnssh.Choosable, defaultQuery string) (rnssh.Choosable, error) {

	if len(choices) == 0 {
		err := fmt.Errorf("there is no running instance.")
		return nil, err
	}

	pecoChoices := make([]peco.Choosable, 0, len(choices))
	for _, c := range choices {
		pecoChoices = append(pecoChoices, c)
	}

	pecoOpt := &peco.PecoOptions{
		OptPrompt: "which do you choose the instance? >",
		OptQuery:  defaultQuery,
	}

	result, err := peco.PecolibWithOptions(pecoChoices, pecoOpt)
	if err != nil || len(result) == 0 {
		err := fmt.Errorf("no select target.")
		return nil, err
	}

	var targetHost rnssh.Choosable
	for _, r := range result {
		if ec2host, ok := r.(rnssh.Choosable); ok {
			targetHost = ec2host
		} else {
			err := fmt.Errorf("this is bug. type is not Choosable: %v", r)
			return nil, err
		}
	}

	return targetHost, nil
}

func getSshTargetType(publicIP, privateIP, nameTag bool) string {

	hostType := os.Getenv(ENV_RNSSH_HOST_TYPE)

	// default host type.
	if hostType == "" {
		hostType = myec2.HOST_TYPE_PUBLIC_IP
	}

	// overwrite by option
	if publicIP {
		hostType = myec2.HOST_TYPE_PUBLIC_IP
	}

	if privateIP {
		hostType = myec2.HOST_TYPE_PRIVATE_IP
	}

	if nameTag {
		hostType = myec2.HOST_TYPE_NAME_TAG
	}

	return hostType
}

func genSshArgs(optSshUser, optIdentityFile string, optPort int, optStrictHostKeyCheckingNo bool, sshUser, sshHost string) []string {
	args := make([]string, 0)
	if optSshUser != "" {
		args = append(args, "-l"+optSshUser)
	}

	if optIdentityFile != "" {
		args = append(args, "-i"+optIdentityFile)
	}

	if optPort > 0 {
		args = append(args, "-p"+strconv.Itoa(optPort))
	}

	if optStrictHostKeyCheckingNo {
		args = append(args, "-oStrictHostKeyChecking=no")
		args = append(args, "-oUserKnownHostsFile=/dev/null")
	}

	if sshUser != "" {
		sshHost = sshUser + "@" + sshHost
	}

	args = append(args, sshHost)

	return args
}
