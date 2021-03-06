package main

import (
	"bytes"
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/reiki4040/cstore"
	"github.com/reiki4040/peco"
)

const (
	RNSSH_EC2_LIST_CACHE_PREFIX = "aws.instances.cache."
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

	w := new(tabwriter.Writer)
	var b bytes.Buffer
	w.Init(&b, 14, 0, 4, ' ', 0)
	if e.TargetType == HOST_TYPE_NAME_TAG {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s", e.InstanceId, e.Name, publicIP, e.PrivateIP)
		w.Flush()
		return string(b.Bytes())
	} else {
		fmt.Fprintf(w, "%s\t%s\t%s", e.InstanceId, e.Name, e.Value())
		w.Flush()
		return string(b.Bytes())
	}
}

func (e *ChoosableEC2) Value() string {
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

type ChoosableEC2s []*ChoosableEC2

func (e ChoosableEC2s) Len() int {
	return len(e)
}

func (e ChoosableEC2s) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e ChoosableEC2s) Less(i, j int) bool {
	return e[i].Name < e[j].Name
}

type Instances struct {
	Instances []*ec2.Instance `json:"ec2_instances"`
}

func NewEC2Handler(m *cstore.Manager) *EC2Handler {
	return &EC2Handler{
		Manager: m,
	}
}

type EC2Handler struct {
	Manager *cstore.Manager
}

func (r *EC2Handler) GetCacheStore(region string) (*cstore.CStore, error) {
	cacheFileName := RNSSH_EC2_LIST_CACHE_PREFIX + region + ".json"
	return r.Manager.New(cacheFileName, cstore.JSON)
}

func (r *EC2Handler) LoadTargetHost(hostType string, region string, reload bool) ([]peco.Choosable, error) {
	var instances []*ec2.Instance
	cacheStore, _ := r.GetCacheStore(region)

	is := Instances{}
	if cErr := cacheStore.GetWithoutValidate(&is); cErr != nil || reload {
		var err error
		instances, err = GetInstances(region)
		if err != nil {
			awsErr := fmt.Errorf("failed get instance: %s", err.Error())
			return nil, awsErr
		}

		is = Instances{Instances: instances}
		if cacheStore != nil {
			err := cacheStore.SaveWithoutValidate(&is)
			if err != nil {
				// only warn message
				fmt.Printf("warn: failed store ec2 list cache: %s\n", err.Error())
			}
		}
	}

	choices := ConvertChoosableList(is.Instances, hostType)
	if len(choices) == 0 {
		err := fmt.Errorf("there is no running instance.")
		return nil, err
	}

	return choices, nil
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

func ConvertChoosableList(instances []*ec2.Instance, targetType string) []peco.Choosable {
	choosableEC2List := make([]*ChoosableEC2, 0, len(instances))
	for _, i := range instances {
		e := convertChoosable(i, targetType)
		if e != nil {
			choosableEC2List = append(choosableEC2List, e)
		}
	}

	sort.Sort(ChoosableEC2s(choosableEC2List))

	choices := make([]peco.Choosable, 0, len(choosableEC2List))
	for _, c := range choosableEC2List {
		choices = append(choices, c)
	}

	return choices
}

func convertChoosable(i *ec2.Instance, targetType string) *ChoosableEC2 {
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

	t := ec2host.Value()
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
