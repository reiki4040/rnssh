package ec2

import (
	"bufio"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/ec2"
)

const (
	ENV_HOME       = "HOME"
	ENV_AWS_REGION = "AWS_REGION"

	RNZOO_DIR_NAME              = ".rnssh"
	RNZOO_EC2_LIST_CACHE_PREFIX = "instances.cache."
)

func GetRnzooDir() string {
	rnzooDir := os.Getenv(ENV_HOME) + string(os.PathSeparator) + RNZOO_DIR_NAME
	return rnzooDir
}

func GetEc2listCachePath(region *aws.Region) string {
	rnzooDir := GetRnzooDir()
	return rnzooDir + string(os.PathSeparator) + RNZOO_EC2_LIST_CACHE_PREFIX + region.Name
}

func CreateRnzooDir() error {
	rnzooDir := GetRnzooDir()

	if _, err := os.Stat(rnzooDir); os.IsNotExist(err) {
		err = os.Mkdir(rnzooDir, 0700)
		if err != nil {
			if !os.IsExist(err) {
				return err
			}
		}
	}

	return nil
}

// get Region from string region name.
func GetRegion(regionName string) (*aws.Region, error) {
	if regionName == "" {
		regionName = os.Getenv(ENV_AWS_REGION)
	}

	region, ok := aws.Regions[strings.ToLower(regionName)]
	if !ok {
		return nil, errors.New(fmt.Sprintf("unknown region name: %s", regionName))
	}

	return &region, nil
}
func GetEC2Array(reload bool, region *aws.Region) ([]ec2.Instance, error) {
	//func ec2list(reload bool, region *aws.Region) {
	var instances []ec2.Instance
	cachePath := GetEc2listCachePath(region)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) || reload {
		auth, err := aws.EnvAuth()
		if err != nil {
			awsErr := fmt.Errorf("failed auth: %s", err.Error())
			return nil, awsErr
		}
		instances, err = GetInstances(auth, region)
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
		instances, err = LoadCache(cachePath)
		if err != nil {
			// only warn message
			fmt.Printf("warn: failed load ec2 list cache: %s, so try load from AWS.\n", err.Error())
			return GetEC2Array(true, region)
		}
	}

	return instances, nil
}

type Instances struct {
	Instances []ec2.Instance `xml:"Instance"`
}

func StoreCache(instances []ec2.Instance, cachePath string) error {
	cacheFile, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer cacheFile.Close()

	w := bufio.NewWriter(cacheFile)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	toXml := Instances{Instances: instances}
	if err := enc.Encode(toXml); err != nil {
		return err
	}

	return nil
}

func LoadCache(cachePath string) ([]ec2.Instance, error) {
	cacheFile, err := os.Open(cachePath)
	if err != nil {
		return nil, err
	}
	defer cacheFile.Close()

	r := bufio.NewReader(cacheFile)
	dec := xml.NewDecoder(r)
	instances := Instances{}
	err = dec.Decode(&instances)
	if err != nil {
		return nil, err
	}

	return instances.Instances, nil
}

func GetInstances(auth aws.Auth, region *aws.Region) ([]ec2.Instance, error) {
	ec2conn := ec2.New(auth, *region)

	resp, err := ec2conn.DescribeInstances(nil, nil)
	if err != nil {
		return nil, err
	}

	if len(resp.Reservations) == 0 {
		return []ec2.Instance{}, nil
	}

	instances := make([]ec2.Instance, 0)
	for _, r := range resp.Reservations {
		for _, i := range r.Instances {
			instances = append(instances, i)
		}
	}

	return instances, nil
}
