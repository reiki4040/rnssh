rnssh
====

[![GitHub release](http://img.shields.io/github/release/reiki4040/rnssh.svg?style=flat-square)][release]
[![MIT License](http://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)][license]

[release]: https://github.com/reiki4040/rnssh/releases
[license]: https://github.com/reiki4040/rnssh/blob/master/LICENSE

easy ssh to ec2 instance.
now MacOS only.

## How to install and settings

- homebrew (recommend)
- download binary

### homebrew

```
brew tap reiki4040/rnssh
brew install rnssh
```

### download archive and set PATH

download rnssh binary file and set PATH

## Settings

- set AWS ENV variables
- ssh config (Optional but recommended)

### set AWS variables (.bashrc, .bash_profile etc...)

    export AWS_ACCESS_KEY_ID=
    export AWS_SECRET_ACCESS_KEY=

    # option: specify default region
    export AWS_REGION=


### ssh config

`vi ~/.ssh/config`

    Host X.X.X.X
      User your_user
      IdentityFile you_key_file

***More useful If you added your ec2 instances to ssh config before using rnssh by yourself.***

## How to use

### run command

    rnssh -i identity_file user@host

you can run `rnssh` (without options `-l`,`-i`) if you added instances to ssh config.

show ec2 instances list. you can filtering.

    Select ssh instance. You can do filtering>
    instance name1 X.X.X.X
    instance name2 X.X.X.Y

choose the instance, then start ssh to the instance.

    instanse $

## More useful

### cache

rnssh does create cache the instances list automatically.
if you update instances, you must be reload with `-f` option.
(launch, start, stop etc...)

without `-f`, rnssh does load from cache file. it is faster than connect to AWS(with `-f`).

### ssh config

if you created ssh config (ex ~/.ssh/config), rnssh can works without `-l`, `-i` options.

    Host <ec2_ipaddress>
         User <ssh_user>
         IdentityFile <to_identity_fie_path>

### filtering

rnssh can filter instances with using arguments

    rnssh web server

already filtered and it is able to modify if you want.

    QUERY>web server
    web server1 X.X.X.X
    web server2 Y.Y.Y.Y

### change default ssh host type with `RNSSH_HOST_TYPE`

if you always rnssh with `-p`(Private IP) or `-n`(Name Tag), RNSSH_HOST_TYPE environment variable will be help.
this variable can change default ssh host type.

valid values are below.

- `public` (default)
- `private`
- `name`

and you can use `-P` `-p` `-n`, when you want to use other ssh host type temporarily.

## TODO

- Migrate to aws-go-sdk
- homebrew
- Test code

## Copyright and LICENSE

Copyright (c) 2015- reiki4040

MIT LICENSE
