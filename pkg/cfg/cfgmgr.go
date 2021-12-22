// Copyright 2020 Platform9 Systems Inc.

package cfg

import (
    "errors"
    "fmt"
    rand "math/rand"
    "reflect"
    "sort"
    "time"

    consul "github.com/hashicorp/consul/api"
    "database/sql"
    _ "github.com/go-sql-driver/mysql"
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
	regionKSUserPrefix := fmt.Sprintf("%s/services/%s/keystone_user", c.RegionKeyPrefix, serviceName)

	// get the password
	password, err := c.getValue(fmt.Sprintf("%s/password", globalKSUserPrefix))
	if err != nil {
		zap.L().Error("Can't get password for serviceName: ", zap.String("serviceName", serviceName))
		password = c.GetRandomPassword()
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

func (c * CfgMgr) GetRandomPassword() string {
    rand.Seed(time.Now().UnixNano())
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

func (c *CfgMgr) CreateDB(serviceName, userName string) (updateConsul bool, err error) {
    dbObject, err := c.getDbObject()
    if err != nil {
        return false, err
    }
    _, err = dbObject.Exec(fmt.Sprintf("SHOW DATABASE LIKE %s", serviceName))
    if err == nil {
        zap.L().Info(fmt.Sprintf("Database %s already exists", serviceName))
        return false, nil
    }
    zap.L().Debug("Creating DB for serviceName ", zap.String("serviceName", serviceName))
    _, err = dbObject.Exec(fmt.Sprintf("CREATE DATABASE %s", serviceName))
    if err != nil {
        zap.L().Error("Error while creating database", zap.Error(err))
        return false, err
    }
    zap.L().Info(fmt.Sprintf("Created DB '%s' successfully", serviceName))
    return true, nil
}

func (c *CfgMgr) CreateGrants(dbName, userName, dbPassword string) (bool, error) {
    dbObject, err := c.getDbObject()
    if err != nil {
        return false, err
    }
    rows, err := dbObject.Query("SELECT @@hostname")
    if err != nil {
        zap.L().Error("Error while getting hostname", zap.Error(err))
        return false, err
    }
    defer rows.Close()
    var hostname string
    count := 0
    for rows.Next() {
        _ = rows.Scan(&hostname)
    }
    hosts := []string{"localhost", "%", hostname}
    for _, hostName := range hosts {
        before_grants := c.getGrants(userName, hostName, dbObject)
        query := fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'%s' IDENTIFIED BY '%s'",
                             dbName, userName, hostName, dbPassword)
        _, _ = dbObject.Exec(query)
        after_grants := c.getGrants(userName, hostName, dbObject)
        if (reflect.DeepEqual(before_grants, after_grants)) {
            count += 1
        }
    }
    if count == len(hosts) {
        return false, nil
    } else {
        return true, nil
    }
}

func (c *CfgMgr) getDbDetails() (string, string, string, string, error) {
    dbserver, err := c.getValue(fmt.Sprintf("%s/keystone/dbserver_key", c.CustomerKeyPrefix))
    if err != nil {
        zap.L().Error("Cannot get dbserver_key from consul store", zap.Error(err))
        return "", "", "", "", err
    }
    host, err := c.getValue(fmt.Sprintf("%s/host", dbserver))
    if err != nil {
        zap.L().Error("Cannot get host key from consul store", zap.Error(err))
        return "", "", "", "", err
    }
    port, err := c.getValue(fmt.Sprintf("%s/port", dbserver))
    if err != nil {
        zap.L().Error("Cannot get port key from consul store", zap.Error(err))
        return "", "", "", "", err
    }
    adminUser, err := c.getValue(fmt.Sprintf("%s/admin_user", dbserver))
    if err != nil {
        zap.L().Error("Cannot get admin_user key from consul store", zap.Error(err))
        return "", "", "", "", err
    }
    adminPass, err := c.getValue(fmt.Sprintf("%s/admin_pass", dbserver))
    if err != nil {
        zap.L().Error("Cannot get admin_pass key from consul store", zap.Error(err))
        return "", "", "", "", err
    }
    return host, port, adminUser, adminPass, nil
}

func (c *CfgMgr) getDbObject() (*sql.DB, error) {
    host, port, adminUser, adminPass, err := c.getDbDetails()
    if err != nil {
        return nil, err
    }
    dbObject, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/", adminUser, adminPass, host, port))
    if err != nil {
        zap.L().Error("Can't connect to MySQL", zap.Error(err))
        return nil, err
    }
    return dbObject, nil
}

func (c *CfgMgr) UpdateConsul(serviceName, userName, dbPassword string) error {
    host, port, _, _, err := c.getDbDetails()
    if err != nil {
        return err
    }
    dbPrefix := fmt.Sprintf("%s/%s/db", c.CustomerKeyPrefix, serviceName)
    ops := consul.KVTxnOps{
        &consul.KVTxnOp{Verb: consul.KVSet, Key: fmt.Sprintf("%s/name", dbPrefix), Value: []byte(serviceName)},
        &consul.KVTxnOp{Verb: consul.KVSet, Key: fmt.Sprintf("%s/password", dbPrefix), Value: []byte(dbPassword)},
        &consul.KVTxnOp{Verb: consul.KVSet, Key: fmt.Sprintf("%s/user", dbPrefix), Value: []byte(userName)},
        &consul.KVTxnOp{Verb: consul.KVSet, Key: fmt.Sprintf("%s/host", dbPrefix), Value: []byte(host)},
        &consul.KVTxnOp{Verb: consul.KVSet, Key: fmt.Sprintf("%s/port", dbPrefix), Value: []byte(port)},
    }
    _, _, _, err = c.ConsulKV.Txn(ops, nil)
    if err != nil {
        zap.L().Error("Can't write db config to Consul", zap.Error(err))
        return err
    }
    return nil
}

func (c *CfgMgr) getGrants(userName, host string, dbObject *sql.DB) []string {
    var grants []string
    var field string
    rows, err := dbObject.Query(fmt.Sprintf("SHOW GRANTS FOR %s@%s", userName, host))
    if err != nil {
        zap.L().Error("Error while getting grants for user")
        return grants
    }
    defer rows.Close()
    for rows.Next() {
        _ = rows.Scan(&field)
    }
    grants = append(grants, field)
    sort.Strings(grants)
    return grants
}
