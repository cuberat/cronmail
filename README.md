The cronmail program sends email for a cron job.

Output from cron jobs to standard output or standard error are sent as
email. However, customization of such emails is limited. The goal of
cronmail is to remove some of those limitations by capturing standard
output and standard error to take over the job of sending email.

## Installation
To get the latest changes and install:

```bash
go get github.com/cuberat/go-cronmail/cronmail
go install -i github.com/cuberat/go-cronmail/cronmail
```

## Getting Started

```bash
Usage: ./cronmail [options] command ...

Options:
  -conf file
    	Configuration file. Defaults to ~/etc/cronmail.conf
  -debug
    	Print out configuration information.
  -from address
    	From address to use for message.
  -listid List-Id
    	List-Id value to insert.
  -outlimit output_limit
    	Maximum size in bytes of output from command to send in email. Defaults
    	to 1 MB.
  -prependcmd
    	Prepend the command-line to the email.
  -server server:port
    	Override SMTP server:port in configuration file
  -subject string
    	Subject of message. Defaults to command line.
  -to addresses
    	Recipients addresses for message. Defaults to the value of the MAILTO environment variable.
  -version
    	Print the version of cronmail.

\# Note that options may start with either "-" or "--".
```
