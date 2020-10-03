
![Go](https://github.com/roopakparikh/go-firkinize/workflows/Go/badge.svg)

# go-firkinize
A go version of firkinize library, also available as a binary that can be used for easy scripting

# Scripting use case
Download the appropriate version of the go-firkinize cli from the github release page

There are only two subcommands avaialble today, more can be added later on.

* get-keystone: will let you query the password for a service (in keystone)
* add-keystone: will add entries for a service in the keystone service catalog but also would create user/password for the service. Password is randomly generated and user name is the name of the service itself.

```
./go-firkinize
A simple utility that hides the complexity associated with Platform9
		config store i.e. consul/vault as of today.

Usage:
  firkinize [command]

Available Commands:
  add-keystone Keystone related commands
  get-keystone Get Keystone related attributes
  help         Help about any command

Flags:
      --consul-host-port string   Where to connect to consul server (default "konsul-x.domain.com:3306")
      --consul-scheme string      Consul API scheme can be http/https/jrpc (default "https")
      --consul-token string       Security token to talk to consul server (default "xxx-xxcxcvxcv")
      --customer-id string        ID of the customer under which it is operating (default "c423f1d3-xxxcvxc")
      --debug                     Enable debug logging
  -h, --help                      help for firkinize
      --region-id string          ID of the region under which it is operating (default "6296dd9c-9c62-4da7-b239-882d004d36bb")

Use "firkinize [command] --help" for more information about a command.
```

Firkinize will let you add keystone service endpoints and will also let you query it.

The CLI takes a lot of parameter, but in deccaxon environment, they are picked up from the corresponding environment variables so in practice you don't need to supply them you can simply do the following:

```
./go-firkinize get-keystone --service-name hagrid
asdfasdfasdf
```

For debugging purposes you can pass in a ``` --debug ``` flag that would spit a lot of information on __stderr__ so that while you debug you can still use the go-firkinize in a shell script and get the output for example.
```
$ HAGRID_PASS=`./go-firkinize --debug get-keystone --service-name hagrid`
2020-10-02T10:21:43.824-0700	DEBUG	cmd/root.go:71	Debug log enabled
2020-10-02T10:21:43.824-0700	DEBUG	cfg/cfgmgr.go:29	Dong consul setup
2020-10-02T10:21:43.824-0700	DEBUG	cfg/cfgmgr.go:52	Consul setup done
2020-10-02T10:21:43.824-0700	DEBUG	cmd/getkeystone.go:24	Get keystone password
$ echo $HAGRID_PASS
asdfadfwfwsdf
```

## get-keystone

Self explanatory

```
./go-firkinize get-keystone --service-name hagrid
asdfasdfasdf
```


## add-keystone

Add service URL to the keystone and also generate password to interact with keystone, you will need to query the password using get-keystone

```
./go-firkinize  add-keystone --service-name hagrid --ingress-suffix hagrid
{"level":"info","ts":1601659793.1173918,"caller":"cfg/cfgmgr.go:84","msg":"service endpoint config successfully"}
{"level":"info","ts":1601659793.1965299,"caller":"cfg/cfgmgr.go:134","msg":"Keystone user config added successfully"}
```

The service-name is obvious, the ingress-suffix is to allow for flexibility to advertise additional suffix for example if you want to advertise qbert/v2 as the suffix instead of just qbert.
