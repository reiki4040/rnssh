rnssh
====

[![GitHub release](http://img.shields.io/github/release/reiki4040/rnssh.svg?style=flat-square)][release]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]

[release]: https://github.com/reiki4040/rnssh/releases
[license]: https://github.com/reiki4040/rnssh/blob/master/LICENSE

easy ssh to ec2 instance / hosts in ssh config file.
now MacOS only.

## How to install and settings

### homebrew (recommend)

```
brew install reiki4040/tap/rnssh
```

### download archive and set PATH

download rnssh binary file and set PATH

## Settings

run `rnssh -init` and save to rnssh config (~/.rnssh/config)

### AWS EC2

- set AWS credentials
- set AWS default region
- ssh config

### set AWS credentials

* credential file (`~/.aws/credentials`)

```
[default]
aws_access_key_id=your_key_id
aws_secret_access_key=your_secret
```

* Environment variable (`~/.bashrc`, `~/.bash_profile`, etc...)

```
export AWS_ACCESS_KEY_ID=
export AWS_SECRET_ACCESS_KEY=
```

### set default AWS region and host type

run `rnssh -init` and save to rnssh config (~/.rnssh/config)

### ssh config

`vi ~/.ssh/config`

```
Host X.X.X.X
  HostName X.X.X.X
  User your_user
  IdentityFile you_key_file
```

[ec2ssh](https://github.com/mirakui/ec2ssh) helps your ssh configuration.
it generate ssh config from EC2.

## How to use

### run command

```
# set ssh config
rnssh

# not set ssh config
rnssh -i identity_file user@query_string
```

you can run `rnssh` (without options `-i` and user@) if you added instances to ssh config.
show ec2 instances list. you can filtering. if you specify query_string, already filtering instances.

```
Select ssh instance. You can do filtering>
instance name1 X.X.X.X
instance name2 X.X.X.Y
```

choose the instance, then start ssh to the instance!

## More useful

### cache

rnssh does create cache the instances list automatically.
if you update instances, you must be reload with `-f` option.
(launch, start, stop etc...)

without `-f`, rnssh does load from cache file. it is faster than connect to AWS(with `-f`).

### filtering

rnssh can filter instances with using arguments

```
rnssh web server
```

already filtered and it is able to modify if you want.

```
QUERY>web server
web server1 X.X.X.X
web server2 Y.Y.Y.Y
```

if you delete character, then show other name instances again.

### [AWS EC2] change default ssh host type with `-init`

if you always rnssh with `-p`(Private IP) or `-n`(Name Tag), you can edit default with `rnssh -init`

host type's valid values are below.

- `public` (default)
- `private`(for VPN/Bastion)
- `name`(need ssh config)

and you can use `-P` `-p` `-n`, when you want to use other ssh host type temporarily.

### switch ssh config / AWS EC2

if you want to use other temporarily, then you can use `-use-ssh-config` and `-use-ec2` option.

## Update version

### homebrew

update & upgrade

```
brew update
brew upgrade rnssh
```

### binary

please replace to new binary.

## TODO

- Test code

## Copyright and LICENSE

Copyright (c) 2015- reiki4040

MIT LICENSE
