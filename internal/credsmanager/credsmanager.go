package credsmanager

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

type Credentials struct {
	User     string
	Domain   string
	Password string
}

func New() *Credentials {
	return &Credentials{}
}

func (c *Credentials) RequestForCredentials() error {
	var (
		user     string
		domain   string
		password string
	)

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter username: ")
	user, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	fmt.Println("Enter domain: ")
	domain, err = reader.ReadString('\n')
	if err != nil {
		return err
	}

	fmt.Println("Enter password: ")
	bytePasswd, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return err
	}

	password = string(bytePasswd)

	c.User = strings.TrimSpace(user)
	c.Password = strings.TrimSpace(password)
	c.Domain = strings.TrimSpace(domain)

	return nil

}
