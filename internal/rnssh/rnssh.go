package rnssh

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/reiki4040/cstore"
	"github.com/reiki4040/peco"
)

const (
	ENV_HOME = "HOME"

	RNSSH_DIR_NAME = ".rnssh"

	HOST_TYPE_PUBLIC_IP  = "public"
	HOST_TYPE_PRIVATE_IP = "private"
	HOST_TYPE_NAME_TAG   = "name"
)

type Choosable interface {
	Choice() string
	Value() string
}

type Choice struct {
	C string
	V string
}

func (c *Choice) Choice() string {
	return c.C
}

func (c *Choice) Value() string {
	return c.V
}

func GetRnsshDir() string {
	rnsshDir := os.Getenv(ENV_HOME) + string(os.PathSeparator) + RNSSH_DIR_NAME
	return rnsshDir
}

func CreateRnsshDir() error {
	rnsshDir := GetRnsshDir()

	if _, err := os.Stat(rnsshDir); os.IsNotExist(err) {
		err = os.Mkdir(rnsshDir, 0700)
		if err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}

	return nil
}

type Config struct {
	Default RnsshConfig
}

func (c *Config) Validate() error {
	return c.Default.Validate()
}

type RnsshConfig struct {
	Name                       string `toml:"profile_name,omitempty"`
	AWSRegion                  string `toml:"aws_region"`
	HostType                   string `toml:"host_type"`
	SshUser                    string `toml:"ssh_user"`
	SshIdentityFile            string `toml:"ssh_identitiy_file"`
	SshPort                    int    `toml:"ssh_port,omitzero"`
	SshStrictHostKeyCheckingNo int    `toml:"ssh_strict_host_key_checking_no,omitzero"`

	//AWSKey                     string `toml:"aws_access_key_id"`
	//AWSSecret                  string `toml:"aws_secret_access_key"`
}

func (c *RnsshConfig) Validate() error {

	if err := HostTypeCheck(c.HostType); err != nil {
		return err
	}

	if err := IdentityFileCheck(c.SshIdentityFile); err != nil {
		return err
	}

	if err := StrictHostKeyCheckingNoCheck(c.SshStrictHostKeyCheckingNo); err != nil {
		return err
	}

	return nil
}

func HostTypeCheck(t string) error {
	switch t {
	case HOST_TYPE_PUBLIC_IP:
		fallthrough
	case HOST_TYPE_PRIVATE_IP:
		fallthrough
	case HOST_TYPE_NAME_TAG:
		fallthrough
	case "":
		return nil
	default:
		return fmt.Errorf("invalid HostType value: %s. allow public, private, name or \"\"(default)")
	}
}

func StrictHostKeyCheckingNoCheck(v int) error {
	switch v {
	case 1:
		fallthrough
	case 0:
		fallthrough
	case -1:
		return nil
	default:
		return fmt.Errorf("invalid StrictHostKeyCheckingNo value: %d. allow 0(off), 1(on), -1(not specify)", v)
	}
}

func IdentityFileCheck(path string) error {
	if path != "" {
		// replace ~ -> home dir
		if i := strings.Index(path, "~"); i == 0 {
			user, err := user.Current()
			if err != nil {
				return fmt.Errorf("can not resolved home dir: %s", err.Error())
			}
			path = user.HomeDir + string(os.PathSeparator) + path[1:]
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("not exists: %s", path)
		}
	}

	return nil
}

func DoConfigWizard(cs *cstore.CStore) error {
	chosenRegion, err := choose("AWS region", "Please select default AWS region", AWSRegionList)
	if err != nil {
		return fmt.Errorf("region choose error:%s", err.Error())
	}

	region := ""
	for _, c := range chosenRegion {
		region = c.Value()
		break
	}

	chosenHostType, err := choose("rnssh host type", "Please select default host type", HostTypeList)
	if err != nil {
		return fmt.Errorf("region choose error:%s", err.Error())
	}

	hostType := ""
	for _, c := range chosenHostType {
		hostType = c.Value()
		break
	}

	c := &Config{
		Default: RnsshConfig{
			AWSRegion: region,
			HostType:  hostType,
		},
	}

	if err := cs.Save(c); err != nil {
		return err
	}

	return nil
}

var (
	AWSRegionList = []Choosable{
		&Choice{C: "ap-northeast-1 (Tokyo)", V: "ap-northeast-1"},
		&Choice{C: "ap-southeast-1 (Singapore)", V: "ap-southeast-1"},
		&Choice{C: "ap-southeast-2 (Sydney)", V: "ap-southeast-2"},
		&Choice{C: "eu-central-1 (Frankfurt)", V: "eu-central-1"},
		&Choice{C: "eu-west-1 (Ireland)", V: "eu-west-1"},
		&Choice{C: "sa-east-1 (Sao Paulo)", V: "sa-east-1"},
		&Choice{C: "us-east-1 (N. Virginia)", V: "us-east-1"},
		&Choice{C: "us-west-1 (N. California)", V: "us-west-1"},
		&Choice{C: "us-west-2 (Oregon)", V: "us-west-2"},
	}

	HostTypeList = []Choosable{
		&Choice{C: "PublicIP (rnssh default)", V: "public"},
		&Choice{C: "PrivateIP (for VPN or bastion)", V: "private"},
		&Choice{C: "Name Tag (need ssh config settings)", V: "name"},
	}
)

func choose(itemName, message string, choices []Choosable) ([]Choosable, error) {
	if len(choices) == 0 {
		err := fmt.Errorf("there is no %s.", itemName)
		return nil, err
	}

	pecoChoices := make([]peco.Choosable, 0, len(choices))
	for _, c := range choices {
		pecoChoices = append(pecoChoices, c)
	}

	pecoOpt := &peco.PecoOptions{
		OptPrompt: fmt.Sprintf("%s >", message),
	}

	result, err := peco.PecolibWithOptions(pecoChoices, pecoOpt)
	if err != nil || len(result) == 0 {
		err := fmt.Errorf("no select %s.", itemName)
		return nil, err
	}

	chosen := make([]Choosable, 0, len(result))
	for _, r := range result {
		if c, ok := r.(Choosable); ok {
			chosen = append(chosen, c)
		}
	}

	return chosen, nil
}

func ask(msg, defaultValue string) (string, error) {
	fmt.Printf("%s[%s]:", msg, defaultValue)
	reader := bufio.NewReader(os.Stdin)

	ans, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("input err:%s", err.Error())
	}

	return ans, nil
}
