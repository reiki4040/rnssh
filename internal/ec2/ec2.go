package ec2

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/reiki4040/rnssh/internal/rnssh"
)

const (
	ENV_AWS_REGION = "AWS_REGION"

	RNSSH_EC2_LIST_CACHE_PREFIX = "aws.instances.cache."

	HOST_TYPE_PUBLIC_IP  = "public"
	HOST_TYPE_PRIVATE_IP = "private"
	HOST_TYPE_NAME_TAG   = "name"
)

type ChoosableEC2 struct {
	InstanceId string
	Name       string
	PublicIP   string
	PrivateIP  string
	TargetType string
}

func (e *ChoosableEC2) Choice() string {
	publicIP := e.PublicIP
	if publicIP == "" {
		publicIP = "NO_PUBLIC_IP"
	}

	if e.TargetType == HOST_TYPE_NAME_TAG {
		return fmt.Sprintf("%s\t%s\t%s\t%s", e.InstanceId, e.Name, publicIP, e.PrivateIP)
	} else {
		return fmt.Sprintf("%s\t%s\t%s", e.InstanceId, e.Name, e.GetSshTarget())
	}
}

func (e *ChoosableEC2) GetSshTarget() string {
	switch e.TargetType {
	case HOST_TYPE_PUBLIC_IP:
		return e.PublicIP
	case HOST_TYPE_PRIVATE_IP:
		return e.PrivateIP
	case HOST_TYPE_NAME_TAG:
		return e.Name
	default:
		return ""
	}
}

// get Region from string region name.
func GetRegion(regionName string) string {
	if regionName == "" {
		regionName = os.Getenv(ENV_AWS_REGION)
	}

	return strings.ToLower(regionName)
}

type Instances struct {
	Instances []*ec2.Instance `json:"ec2_instances"`
}

func DefaultEC2Handler() *EC2Handler {
	return &EC2Handler{
		CacheDirPath: rnssh.GetRnsshDir(),
	}
}

type EC2Handler struct {
	CacheDirPath string
}

func (r *EC2Handler) GetEc2listCachePath(region string) string {
	return r.CacheDirPath + string(os.PathSeparator) + RNSSH_EC2_LIST_CACHE_PREFIX + region + ".json"
}

func (r *EC2Handler) LoadTargetHost(hostType string, region string, reload bool) ([]rnssh.Choosable, error) {
	var instances []*ec2.Instance
	cachePath := r.GetEc2listCachePath(region)

	if _, err := os.Stat(cachePath); os.IsNotExist(err) || reload {
		var err error
		instances, err = GetInstances(region)
		if err != nil {
			awsErr := fmt.Errorf("failed get instance: %s", err.Error())
			return nil, awsErr
		}

		if err != nil {
			awsErr := fmt.Errorf("failed get instance: %s", err.Error())
			return nil, awsErr
		}

		err = StoreCache(instances, cachePath)
		if err != nil {
			// only warn message
			fmt.Printf("warn: failed store ec2 list cache: %s\n", err.Error())
		}
	} else {
		var err error
		instances, err = LoadCache(cachePath)
		if err != nil {
			// only warn message
			fmt.Printf("warn: failed load ec2 list cache: %s, so try load from AWS.\n", err.Error())

			instances, err = GetInstances(region)
			if err != nil {
				awsErr := fmt.Errorf("failed get instance: %s", err.Error())
				return nil, awsErr
			}
		}
	}

	choices := ConvertChoosableList(instances, hostType)
	if len(choices) == 0 {
		err := fmt.Errorf("there is no running instance.")
		return nil, err
	}

	return choices, nil
}

func StoreCache(instances []*ec2.Instance, cachePath string) error {
	cacheFile, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	w := bufio.NewWriter(cacheFile)
	enc := json.NewEncoder(w)
	//enc.Indent("", "  ")
	toJson := Instances{Instances: instances}
	if err := enc.Encode(toJson); err != nil {
		return err
	}

	return nil
}

func LoadCache(cachePath string) ([]*ec2.Instance, error) {
	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return nil, err
	}
	defer cacheFile.Close()

	r := bufio.NewReader(cacheFile)
	dec := json.NewDecoder(r)
	instances := Instances{}
	err = dec.Decode(&instances)
	if err != nil {
		return nil, err
	}

	return instances.Instances, nil
}

func GetInstances(region string) ([]*ec2.Instance, error) {
	cli := ec2.New(session.New(), &aws.Config{Region: aws.String(region)})

	resp, err := cli.DescribeInstances(nil)
	if err != nil {
		return nil, err
	}

	if len(resp.Reservations) == 0 {
		return []*ec2.Instance{}, nil
	}

	instances := make([]*ec2.Instance, 0)
	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, i)
		}
	}

	return instances, nil
}

func ConvertChoosableList(instances []*ec2.Instance, targetType string) []rnssh.Choosable {
	choices := make([]rnssh.Choosable, 0, len(instances))
	for _, i := range instances {
		c := convertChoosable(i, targetType)
		if c != nil {
			choices = append(choices, c)
		}
	}

	return choices
}

func convertChoosable(i *ec2.Instance, targetType string) rnssh.Choosable {
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
		InstanceId: convertNilString(ins.InstanceId),
		Name:       nameTag,
		PublicIP:   convertNilString(ins.PublicIpAddress),
		PrivateIP:  convertNilString(ins.PrivateIpAddress),
		TargetType: targetType,
	}

	t := ec2host.GetSshTarget()
	if t == "" {
		return nil
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
