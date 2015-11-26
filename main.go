package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	flag "github.com/dotcloud/docker/pkg/mflag"

	"github.com/reiki4040/cstore"
	"github.com/reiki4040/peco"
)

const (
	Usage = `rnssh - easy ssh login to EC2.

usage:

  rnssh [-f] [-p] [-s] [user@]query strings ...
  rnssh --init

options:
  -f: reload ec2 instances infomaion. connect to AWS.
      you have to specify this option after you modified instances.
	  (launch, start, stop, modify name, etc...)

  -P: use Public IP address. this is default ssh host type.
  -p: use Private IP address. for VPN/Direct connect.
  -n: use Name tag.
      this option for ssh config that Host named by ec2 Name tag.

  -r: target region. you can set default by --init (~/.rnssh/config)

  -s: show ssh command string that would be run. (debug)

  --init: start wizard for default setting AWS region and rnssh host type.
          and save to config file (~/.rnssh/config)

options for ssh:
  -l: ssh user.
  -i: identity file path.
  --port: ssh port.
  --strict-host-key-checking-no: suppress host key checking.
                                 1: using 'ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'
                                 0: OFF
                                -1: Default(OFF)

options for help:
  -h: show this usage.
  -v: show version.

args:
  query string...: filtering ec2 instances list.
`

	ENV_AWS_REGION      = "AWS_REGION"
	ENV_RNSSH_HOST_TYPE = "RNSSH_HOST_TYPE"

	ENV_HOME = "HOME"

	RNSSH_DIR_NAME = ".rnssh"
)

type CommandOption struct {
	Reload                  bool
	Region                  string
	PrivateIP               bool
	PublicIP                bool
	NameTag                 bool
	SshUser                 string
	IdentityFile            string
	Port                    int
	StrictHostKeyCheckingNo int
}

func (o *CommandOption) Validate() error {
	if err := duplicateHostTypeOption(o.PublicIP, o.PrivateIP, o.NameTag); err != nil {
		return err
	}

	if err := IdentityFileCheck(o.IdentityFile); err != nil {
		return err
	}

	if err := StrictHostKeyCheckingNoCheck(o.StrictHostKeyCheckingNo); err != nil {
		return err
	}

	return nil
}

func duplicateHostTypeOption(public, private, name bool) error {
	if public && private || private && name || public && name {
		return fmt.Errorf("duplicate specify option -P/-p/-n. please spcify only one.")
	}

	return nil
}

type RnsshOption struct {
	Reload                  bool
	Region                  string
	HostType                string
	SshUser                 string
	IdentityFile            string
	Port                    int
	StrictHostKeyCheckingNo int
}

var (
	version   string
	hash      string
	builddate string

	show_version bool
	show_usage   bool
	initWizard   bool
	showCommand  bool

	// command option
	opt = &CommandOption{}
)

func init() {
	flag.BoolVar(&show_version, []string{"v", "-version"}, false, "show version.")
	flag.BoolVar(&show_usage, []string{"h", "-help"}, false, "show this usage.")
	flag.BoolVar(&initWizard, []string{"-init"}, false, "run initial configuration wizard.")

	flag.BoolVar(&opt.Reload, []string{"f", "-force"}, false, "reload ec2 (force connect to AWS)")
	flag.BoolVar(&opt.PublicIP, []string{"P", "-public-ip"}, false, "ssh with EC2 Public IP")
	flag.BoolVar(&opt.PrivateIP, []string{"p", "-private-ip"}, false, "ssh with EC2 Private IP")
	flag.BoolVar(&opt.NameTag, []string{"n", "-name-tag"}, false, "ssh with EC2 Name tag")
	flag.BoolVar(&showCommand, []string{"s", "-show-command"}, false, "show ssh command that will do (debug)")

	flag.StringVar(&opt.Region, []string{"r", "-region"}, "", "specify region")
	flag.StringVar(&opt.SshUser, []string{"l", "-user"}, "", "specify ssh user")
	flag.StringVar(&opt.IdentityFile, []string{"i", "-identity-file"}, "", "specify ssh identity file")
	flag.IntVar(&opt.Port, []string{"-port"}, 0, "specify ssh port")
	flag.IntVar(&opt.StrictHostKeyCheckingNo, []string{"-strict-host-key-checking-no"}, -1, "suppress host key checking. 1: ON, 0: OFF, -1: default(OFF)")

	flag.Parse()
}

func showVersion() {
	fmt.Printf("%s (%s) built:%s\n", version, hash, builddate)
}

func usage() {
	fmt.Printf("%s\n", Usage)
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

	err := opt.Validate()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	m, err := cstore.NewManager("rnssh", getRnsshDir())
	if err != nil {
		fmt.Printf("can not create rnssh dir: %s\n", err.Error())
		os.Exit(1)
	}

	cs, err := m.New("config", cstore.TOML)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	if initWizard {
		if err := DoConfigWizard(cs); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println("saved rnssh config.")
			os.Exit(0)
		}
	}

	conf := Config{}
	err = cs.Get(&conf)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	rOpt := mergeConfig(&conf.Default, *opt)
	if rOpt.Region == "" {
		fmt.Println("region is empty. please specify by region option (-r) or AWS_REGION envirnment variable.")
		os.Exit(1)
	}

	sshArgs, err := chooseAndGenSshArgs(rOpt, flag.Args(), m)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

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

// merge option, config, ENV
// priority [high] option > config > ENV [low]
func mergeConfig(conf *RnsshConfig, opt CommandOption) *RnsshOption {

	region := os.Getenv(ENV_AWS_REGION)
	if opt.Region != "" {
		region = opt.Region
	} else {
		if conf.AWSRegion != "" {
			region = conf.AWSRegion
		}
	}

	hostType := os.Getenv(ENV_RNSSH_HOST_TYPE)
	optHostType := getSshTargetType(opt.PublicIP, opt.PrivateIP, opt.NameTag)
	if optHostType != "" {
		hostType = optHostType
	} else {
		if conf.HostType != "" {
			hostType = conf.HostType
		}
	}

	sshUser := conf.SshUser
	if opt.SshUser != "" {
		sshUser = opt.SshUser
	}

	identityFile := conf.SshIdentityFile
	if opt.IdentityFile != "" {
		identityFile = opt.IdentityFile
	}

	port := conf.SshPort
	if opt.Port > 0 {
		port = opt.Port
	}

	strictHostKeyCheckingNo := conf.SshStrictHostKeyCheckingNo
	if opt.StrictHostKeyCheckingNo != -1 {
		strictHostKeyCheckingNo = opt.StrictHostKeyCheckingNo
	}

	return &RnsshOption{
		Reload:       opt.Reload,
		Region:       region,
		HostType:     hostType,
		SshUser:      sshUser,
		IdentityFile: identityFile,
		Port:         port,
		StrictHostKeyCheckingNo: strictHostKeyCheckingNo,
	}
}

func chooseAndGenSshArgs(rOpt *RnsshOption, cmdArgs []string, manager *cstore.Manager) ([]string, error) {

	// support user@host format
	sshUser, hostname, err := getSshUserAndHostname(strings.Join(cmdArgs, " "))
	if err != nil {
		return nil, fmt.Errorf("%s\n", err.Error())
	}

	hostType := HOST_TYPE_PUBLIC_IP
	if rOpt.HostType != "" {
		hostType = rOpt.HostType
	}

	handler := NewEC2Handler(manager)
	choosableList, err := handler.LoadTargetHost(hostType, rOpt.Region, rOpt.Reload)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	if len(choosableList) == 0 {
		fmt.Printf("there is no instance. not running %s\n", rOpt.Region)
		os.Exit(1)
	}

	// show ec2 instances and choose intactive
	targetHosts, err := peco.Choose("server", "which servers connect with ssh?", hostname, choosableList)
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	}

	l := len(targetHosts) - 1
	targetHost := targetHosts[l]
	sshHost := targetHost.Value()
	sshArgs := genSshArgs(rOpt.SshUser, rOpt.IdentityFile, rOpt.Port, rOpt.StrictHostKeyCheckingNo, sshUser, sshHost)

	return sshArgs, nil
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

func getSshTargetType(publicIP, privateIP, nameTag bool) string {

	// overwrite by option
	if publicIP {
		return HOST_TYPE_PUBLIC_IP
	}

	if privateIP {
		return HOST_TYPE_PRIVATE_IP
	}

	if nameTag {
		return HOST_TYPE_NAME_TAG
	}

	return ""
}

func genSshArgs(optSshUser, optIdentityFile string, optPort, optStrictHostKeyCheckingNo int, sshUser, sshHost string) []string {
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

	if optStrictHostKeyCheckingNo == 1 {
		args = append(args, "-oStrictHostKeyChecking=no")
		args = append(args, "-oUserKnownHostsFile=/dev/null")
	}

	if sshUser != "" {
		sshHost = sshUser + "@" + sshHost
	}

	args = append(args, sshHost)

	return args
}

func getRnsshDir() string {
	rnsshDir := os.Getenv(ENV_HOME) + string(os.PathSeparator) + RNSSH_DIR_NAME
	return rnsshDir
}
