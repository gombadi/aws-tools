# Gombadi AWS Tools

This repo contains code that Sitback has created for its technical operations.
The code was created to ease the daily work burden and also to upskill
with Go Language. Enjoy

The code in this repo has been updated to use the latest available from AWS
https://github.com/aws/aws-sdk-go

All the code in this repo will use AWS credentials from the environment.

## Installing

Simply use go get to download the code
    $ go get github.com/gombadi/aws-tools

To install all applications change to the new directory and run
    $ for d in $(ls -1 -d awsgo-*); do (cd $d && go install); done




The following applications are available at the moment:

## awsgo-autostop

This simple program will search for any instances that have a tag of autostop
and are in state running. If any are found then the instance is stopped.
This program can be called from a cronjob to shutdown instances
that do not need to be running 24x7 such as staging servers.



## awsgo-asgservers

This simple program will display the private ip addresses of any instances
in the auto scaling group. It is useful to get the internal ip address
if you need to connect to all servers in the group.
If no auto scaling group name is given then all auto scale group names
are displayed.




## awsgo-describe-instances

This simple program will display basic info about the instances in the region.



## awsgo-snapshot-instance
## awsgo-ami-cleanup

These two programs will allow you to create an AMI for an instance and then
automatically cleanup old AMI's after a few days. This is useful if you
want to create a snapshot of your instances for backup purposes.
From cron first run awsgo-snapshot-instance -

14 03 * * * username . ${HOME}/.aws/credentials && /usr/local/bin/awsgo-snapshot-instance -a

This will snapshot all instances that have a tag named autobkup in the account.

24 03 * * * username . ${HOME}/.aws/credentials && /usr/local/bin/awsgo-ami-cleanup -a 2

This will remove any old AMI's and associated snapshots that are older than 2 days.




## awsgo-describe-elb-ssl

This simple program will display information, including expiry date, for all
ssl certificates stored in IAM and used by load balancers. The output is in
csv format and can be directed to a file.



## awsgo-reserved-report

This simple program will output, in csv format, all the EC2 and RDS reserved instances
for the account. It can be used to check on which reserved instances are going to expire.



