# Gombadi AWS Tools

This repo contains code that Sitback has created for its technical operations.
The code was created to ease the daily work burden and also to upskill
with Go Language. Enjoy

The code in this repo has been updated to use the latest available from AWS
https://github.com/aws/aws-sdk-go

All the code in this repo will use AWS credentials from the environment.

## Installing

Simply use go get to download the code:

    $ go get github.com/gombadi/aws-tools

Dependencies:

    $ go get github.com/mitchellh/cli

    $ go get github.com/aws/aws-sdk-go/...



The following sub applications are available at the moment:

```
Usage: awsgo-tools [--version] [--help] <command> [<args>]

Available commands are:
    asgservers    Display auto scale ip addresses
    autostop      Auto stop tagged instances
    iamssl        IAM SSL CSV Output
```


