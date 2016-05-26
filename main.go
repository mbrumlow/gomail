package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/smtp"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type configuration struct {
	Sender   string
	Host     string
	Port     int
	Username string
	Password string
}

func main() {

	var configFile = flag.String("c", "", "Configuration file.")
	var subject = flag.String("subject", "", "Subject of the e-mail.")
	var sendFile = flag.String("body", "", "File containing the body of the e-mail.")
	var recipients = flag.String("to", "", "Comma separated recipients list.")

	flag.Parse()

	if *recipients == "" {
		fatalF("No recipients specified.\n")
	}

	config, err := loadConfig(*configFile)
	if err != nil {
		fatalF("%v\n", err.Error())
	}

	body, shouldClose, err2 := bodyReader(*sendFile)
	if err2 != nil {
		fatalF("%v\n", err2.Error())
	}
	if shouldClose {
		defer body.Close()
	}

	server := fmt.Sprintf("%v:%v", config.Host, config.Port)
	c, err := smtp.Dial(fmt.Sprintf(server))
	if err != nil {
		fatalF("Failed to connect to server `%v`: %v\n", server, err.Error())
	}
	defer func() {
		if err = c.Quit(); err != nil {
			fatalF("Failed to close client: %v\n", err.Error())
		}
	}()

	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         config.Host,
	}

	if err := c.StartTLS(tlsconfig); err != nil {
		fatalF("Failed to start secure connection: %v\n", err.Error())
	}

	auth := loginAuth(config.Username, config.Password)
	if err := c.Auth(auth); err != nil {
		fatalF("Failed to authenticate with server: %v\n", err.Error())
	}

	if err := c.Mail(config.Sender); err != nil {
		fatalF("Failed to set sender: %v\n", err.Error())
	}

	rcpts := strings.Split(*recipients, ",")
	for _, r := range rcpts {
		r := strings.TrimSpace(r)
		if err := c.Rcpt(r); err != nil {
			fatalF("Failed to set recipient '%v': %v\n", r, err.Error())
		}
	}

	wc, err := c.Data()
	if err != nil {
		fatalF("Failed to issue data command: %v\n", err.Error())
	}

	if err := sendMail(*subject, wc, body); err != nil {
		fatalF("%v\n", err.Error())
	}

	err = wc.Close()
	if err != nil {
		fatalF("Failed to close data: %v\n", err.Error())
	}

}

func sendMail(subject string, out io.Writer, in io.Reader) error {

	if subject != "" {
		_, err := fmt.Fprintf(out, "Subject: %v\n\n", strings.TrimSpace(subject))
		if err != nil {
			return fmt.Errorf("Failed to send subject %v", err.Error())
		}
	}

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("Failed to send body: %v", err.Error())
	}

	return nil
}

func loadConfig(configFile string) (configuration, error) {

	if configFile == "" {
		usr, err := user.Current()
		if err != nil {
			return configuration{}, fmt.Errorf("Failed to find users's configuration file: %v", err.Error())
		}
		configFile = filepath.Join(usr.HomeDir, ".config", "gomail", "config.json")
	}

	cf, err := os.Open(configFile)
	if err != nil {
		return configuration{}, fmt.Errorf("Failed to open configuration file at `%v`: %v", configFile, err.Error())
	}
	defer cf.Close()

	c := configuration{}

	if err := json.NewDecoder(cf).Decode(&c); err != nil {
		return configuration{}, fmt.Errorf("Failed to read configuration file at `%v`: %v", configFile, err.Error())
	}

	return c, nil
}

func bodyReader(sendFile string) (io.ReadCloser, bool, error) {

	if sendFile == "" {
		return os.Stdin, false, nil
	}

	file, err := os.Open(sendFile)
	if err != nil {
		return nil, false, fmt.Errorf("Failed to open body file at `%v`: %v", sendFile, err.Error())
	}

	return file, true, nil
}

// AUTH LOGIN support -- office365.com support.

type authLogin struct {
	username string
	password string
	host     string
}

var unexpectedSE = fmt.Errorf("unexpected server challenge")

func loginAuth(username, password string) authLogin {
	return authLogin{username: username, password: password}
}

func (a authLogin) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a authLogin) Next(fromServer []byte, more bool) ([]byte, error) {

	command := strings.TrimSpace(string(fromServer))
	command = strings.ToUpper(command)

	if more {
		switch command {
		case "USERNAME:":
			return []byte(a.username), nil
		case "PASSWORD:":
			return []byte(a.password), nil
		default:
			return nil, unexpectedSE
		}

	}

	return nil, nil
}

func fatalF(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}
