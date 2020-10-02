// Copyright 2020 Platform9 Systems Inc.

package cfg

import (
	"errors"
	"fmt"
	rand "math/rand"

	consul "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// ErrorNotFound signifies absence of SSO configuration
var ErrorNotFound = errors.New("Configuration not found")

// CfgMgr holds consul KV and other keys to interact with the consul server
type CfgMgr struct {
	ConsulKV          *consul.KV
	CustomerID        string
	RegionID          string
	CustomerKeyPrefix string
	RegionKeyPrefix   string
	ServicesKeyPrefix string
}

// Setup reads the configuration from the Consul directly (under decco-axxon)
func Setup(consulHostPort, consulScheme, consulToken, customerID, regionID string) (*CfgMgr, error) {
	zap.L().Debug("Dong consul setup")
	// Get a new client
	client, err := consul.NewClient(&consul.Config{
		Address: consulHostPort,
		Scheme:  consulScheme,
		Token:   consulToken,
	})

	if err != nil {
		zap.L().Info("Consul client creation failed")
		return nil, err
	}
	customerKeyPrefix := fmt.Sprintf("customers/%s", customerID)
	regionKeyPrefix := fmt.Sprintf("%s/regions/%s", customerKeyPrefix, regionID)
	servicesKeyPrefix := fmt.Sprintf("%s/services", regionKeyPrefix)

	consulConfig := &CfgMgr{ConsulKV: client.KV(),
		CustomerID:        customerID,
		RegionID:          regionID,
		CustomerKeyPrefix: customerKeyPrefix,
		RegionKeyPrefix:   regionKeyPrefix,
		ServicesKeyPrefix: servicesKeyPrefix,
	}
	zap.L().Debug("Consul setup done")

	// Get a handle to the KV API
	return consulConfig, nil
}

// AddKeystoneEndpoint sets the keystone endpoint configuration
func (c *CfgMgr) AddKeystoneEndpoint(serviceName string, serviceUrlSuffix string) error {
	zap.L().Debug("Creating keystone endpoint for serviceName ", zap.String("serviceName", serviceName))

	fqdnKey := fmt.Sprintf("%s/fqdn", c.CustomerKeyPrefix)
	fqdnValue, err := c.getValue(fqdnKey)
	if err != nil {
		return err
	}

	serviceEndpointKey := fmt.Sprintf("%s/keystone/endpoints/%s/%s", c.CustomerKeyPrefix, c.RegionKeyPrefix, serviceName)
	serviceEndpointInternalURL := fmt.Sprintf("%s/internal_url", serviceEndpointKey)
	serviceEndpointAdminURL := fmt.Sprintf("%s/admin_url", serviceEndpointKey)
	serviceEndpointType := fmt.Sprintf("%s/type", serviceEndpointKey)
	serviceEndpointValue := fmt.Sprintf("https://%s/%s", fqdnValue, serviceUrlSuffix)

	ops := consul.KVTxnOps{
		&consul.KVTxnOp{Verb: consul.KVSet, Key: serviceEndpointInternalURL, Value: []byte(serviceEndpointValue)},
		&consul.KVTxnOp{Verb: consul.KVSet, Key: serviceEndpointAdminURL, Value: []byte(serviceEndpointValue)},
		&consul.KVTxnOp{Verb: consul.KVSet, Key: serviceEndpointType, Value: []byte(serviceName)},
	}
	_, _, _, err = c.ConsulKV.Txn(ops, nil)
	if err != nil {
		zap.L().Error("Can't write service endpoint config to Consul, please retry later")
		return err
	}
	zap.L().Info("service endpoint config successfully")
	return nil
}

func (c *CfgMgr) GetKeystonePassword(serviceName string) (string, error) {
	// get the password
	globalKSUserPrefix := fmt.Sprintf("%s/keystone/users/%s", c.CustomerKeyPrefix, serviceName)
	password, err := c.getValue(fmt.Sprintf("%s/password", globalKSUserPrefix))
	if err != nil {
		zap.L().Error("Can't get password for serviceName: ", zap.String("serviceName", serviceName))
		return "", err
	}
	return password, nil

}

//  AddKeystoneUser
func (c *CfgMgr) AddKeystoneUser(serviceName string) error {
	zap.L().Debug("Creating keystone user for serviceName ", zap.String("serviceName", serviceName))

	globalKSUserPrefix := fmt.Sprintf("%s/keystone/users/%s", c.CustomerKeyPrefix, serviceName)
	regionKSUserPrefix := fmt.Sprintf("%s/services/%s/keystne_user/", c.RegionKeyPrefix, serviceName)

	// get the password
	password, err := c.getValue(fmt.Sprintf("%s/password", globalKSUserPrefix))
	if err != nil {
		zap.L().Error("Can't get password for serviceName: ", zap.String("serviceName", serviceName))
		password = c.getRandomPassword()
	}

	prefixes := []string{globalKSUserPrefix, regionKSUserPrefix}
	ops := consul.KVTxnOps{}

	for _, prefix := range prefixes {
		emailKey := fmt.Sprintf("%s/email", prefix)
		ops = append(ops, &consul.KVTxnOp{Verb: consul.KVSet, Key: emailKey, Value: []byte(serviceName)})
		passwordKey := fmt.Sprintf("%s/password", prefix)
		ops = append(ops, &consul.KVTxnOp{Verb: consul.KVSet, Key: passwordKey, Value: []byte(password)})

		projectKey := fmt.Sprintf("%s/project", prefix)
		ops = append(ops, &consul.KVTxnOp{Verb: consul.KVSet, Key: projectKey, Value: []byte("services")})
		roleKey := fmt.Sprintf("%s/role", prefix)
		ops = append(ops, &consul.KVTxnOp{Verb: consul.KVSet, Key: roleKey, Value: []byte("admin")})
	}

	_, _, _, err = c.ConsulKV.Txn(ops, nil)
	if err != nil {
		zap.L().Error("Can't write keystone users config to Consul, please retry later")
		return err
	}
	zap.L().Info("Keystone user config added successfully")
	return nil
}

func (c *CfgMgr) getValue(key string) (string, error) {

	keyVal, _, err := c.ConsulKV.Get(key, nil)
	if keyVal != nil && keyVal.Value != nil && string(keyVal.Value) != "" {
		return string(keyVal.Value), nil
	} else if err != nil {
		zap.L().Info("error fetching the values")
		return "", err
	}
	return "", fmt.Errorf("Key value for key %s not found", key)
}

func (c *CfgMgr) getRandomPassword() string {
	digits := "0123456789"
	lowerChars := "abcdefghijklmnopqrstuvwxyz"
	upperChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	allChars := lowerChars + upperChars + digits

	const max = 16
	out := make([]byte, max)
	for i := 0; i < max; i++ {
		out[i] = allChars[rand.Intn(len(allChars))]
	}
	return string(out)
}
