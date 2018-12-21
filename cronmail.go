// Copyright (c) 2018 Don Owens <don@regexguy.com>.  All rights reserved.
//
// This software is released under the BSD license:
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
//
//  * Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
//
//  * Redistributions in binary form must reproduce the above
//    copyright notice, this list of conditions and the following
//    disclaimer in the documentation and/or other materials provided
//    with the distribution.
//
//  * Neither the name of the author nor the names of its
//    contributors may be used to endorse or promote products derived
//    from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
// FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
// COPYRIGHT OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
// INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
// HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED
// OF THE POSSIBILITY OF SUCH DAMAGE.


package main

import (
    "flag"
    "fmt"
    "github.com/cuberat/go-ini/ini"
    "net/mail"
    "net/smtp"
    "os"
    "os/exec"
    "os/user"
    "strings"
)

func main() {
    var (
        conf_file string
        smtp_server string
        subject string
        to, from string
        list_id string
    )

    flag.StringVar(&conf_file, "conf", "", "Configuration `file`. Defaults to ~/etc/cronmail.conf")
    flag.StringVar(&smtp_server, "server", "", "Override SMTP `server` in configuration file")
    flag.StringVar(&subject, "subject", "", "Subject of message. Defaults to command line.")
    flag.StringVar(&from, "from", "", "From `address` to use for message.")
    flag.StringVar(&to, "to", "", "Recipients `addresses` for message. Defaults to the value of the MAILTO environment variable.")
    flag.StringVar(&list_id, "listid", "", "`List-Id` value to insert.")

    // FIXME: add info about conf file
    flag.Usage = func() {
        fmt.Fprintf(flag.CommandLine.Output(),
            "Usage: %s [options] command ...\n\nOptions:\n", os.Args[0])
        flag.PrintDefaults()
    }

    flag.Parse()

    args := flag.Args()

    if subject == "" {
        subject = strings.Join(args, " ")
    }

    if conf_file == "" {
        me, _ := user.Current()
        conf_file = fmt.Sprintf("%s/etc/cronmail.conf", me.HomeDir)
    }

    conf_data, err := load_conf(conf_file, "cronmail")
    if err != nil {
        fmt.Fprintf(os.Stderr, "failed to load configuration file %s: %s\n",
            conf_file, err)
        os.Exit(-1)
    }

    out_str, err := run_cmd(args)
    if err != nil {
        fmt.Fprintf(os.Stderr, "cmd failed (%s) : %s\n", strings.Join(args, " "), err)
        os.Exit(-1)
    }

    err = send_mail(conf_data, smtp_server, subject, from, to, out_str, list_id)
    if err != nil {
        fmt.Fprintf(os.Stderr, "couldn't send email: %s\n\nOutput:\n%s\n",
            err, out_str)
        os.Exit(-1)
    }
}

func send_mail(conf_data map[string]string, smtp_server, subject, from, to,
    body, list_id string) (error) {

    var (
        auth_user, auth_passwd string
        ok bool
        auth smtp.Auth
        smtp_hostname string
        extra_headers string
    )

    if smtp_server == "" {
        smtp_server, ok = conf_data["server"]
        if !ok {
            smtp_server = "localhost:25"
        }
    }

    if to == "" {
        to, ok = os.LookupEnv("MAILTO")
        if !ok {
            to, ok = conf_data["mailto"]
            if !ok {
                me, _ := user.Current()
                to = me.Username
            }
        }
    }

    if from == "" {
        from, ok = conf_data["mailfrom"]
        if !ok {
            me, _ := user.Current()
            from = me.Username
        }
    }

    from_addr_obj, err := mail.ParseAddress(from)
    if err != nil {
        if strings.Contains(fmt.Sprintf("%s", err), "no angle-addr") {
            from_addr_obj, err = mail.ParseAddress("<" + from + ">")
        }

        if err != nil {
            return fmt.Errorf("couldn't parse from address: %s", err)
        }
    }

    to_addr_objs, err := mail.ParseAddressList(to)
    if err != nil {
        return fmt.Errorf("couldn't parse to address(es): %s", err)
    }

    to_addrs := make([]string, 0, len(to_addr_objs))
    for _, addr := range to_addr_objs {
        to_addrs = append(to_addrs, addr.Address)
    }

    auth_user, ok = conf_data["auth_user"]
    if ok {
        auth_passwd, ok = conf_data["auth_passwd"]
    }

    if list_id != "" {
        extra_headers += fmt.Sprintf("List-Id: %s\r\n", list_id)
    }

    headers := fmt.Sprintf("%sFrom: %s\r\nTo: %s\r\nSubject: %s\r\n",
        extra_headers, from, to, subject)

    msg_str := fmt.Sprintf("%s\r\n%s", headers, body)

    idx := strings.Index(smtp_server, ":")
    if idx >= 0 {
        smtp_hostname = smtp_server[0:idx]
    } else {
        smtp_hostname = smtp_server
    }

    if auth_user != "" {
        auth = smtp.PlainAuth("", auth_user, auth_passwd, smtp_hostname)
        // auth = smtp.CRAMMD5Auth(auth_user, auth_passwd)
    }

    err = smtp.SendMail(smtp_server, auth, from_addr_obj.Address, to_addrs, []byte(msg_str))

    return err
}

func run_cmd(args []string) (string, error) {
    var cmd_args []string

    if len(args) == 0 {
        return "", fmt.Errorf("no command provided to run")
    }

    cmd_name := args[0]
    cmd_path, err := exec.LookPath(cmd_name)
    if err != nil {
        return "", fmt.Errorf("error running command %s: %s", cmd_name, err)
    }

    if len(args) > 1 {
        cmd_args = args[1:]
    } else {
        cmd_args = make([]string, 0)
    }

    cmd := exec.Command(cmd_path, cmd_args...)

    writer := new(strings.Builder)

    cmd.Stdout = writer
    cmd.Stderr = writer

    err = cmd.Run()
    if err != nil {
        return "", err
    }

    return writer.String(), nil
    
}

func load_conf(conf_file, section string) (map[string]string, error) {
    conf_data, err := ini.LoadFromFile(conf_file)
    if err != nil {
        return nil, err
    }

    if section == "" {
        section = "cronmail"
    }
    cronmail_conf, ok := conf_data[section]
    if !ok {
        return nil, fmt.Errorf("couldn't find '%s' section in %s", section, conf_file)
    }

    return cronmail_conf, nil
}
