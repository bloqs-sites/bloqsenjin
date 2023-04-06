package auth

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
)

var (
	hostname string

	email_regex *regexp.Regexp
)

const (
	email_regex_str = "(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])+)\\])"
)

func init() {
	regex, err := regexp.Compile(email_regex_str)
	if err != nil {
		panic(err)
	}

	email_regex = regex

	hostname, err = os.Hostname()
	if err != nil {
		panic(err)
	}
}

func verifyEmail(email string) error {
	valid := email_regex.Match([]byte(email))

	if !valid {
		return fmt.Errorf("The email `%s` has an invalid format", email)
	}

	email_domain := strings.Split(email, "@")[1]

	if err := helpers.ValidateDomain(email_domain); err != nil {
		return err
	}

	mxr, err := net.LookupMX(email_domain)

	if err != nil {
		return err
	}

	sort.SliceStable(mxr, func(i, j int) bool {
		return mxr[i].Pref < mxr[j].Pref
	})

	ch, n := make(chan errorCloseClosure, len(mxr)+1), 0

	v, t := helpers.GetDomainsListType()
	for _, i := range mxr {
		switch t {
		case helpers.DOMAINS_BLACKLIST:
			for _, d := range v {
				if d == i.Host {
					return fmt.Errorf("The email `%s` uses a blacklisted domain for a MX record", email)
				}
			}
		}
		go smtpVerify(ch, email, i.Host)
	}
	go smtpVerify(ch, email, email_domain)

	for {
		select {
		case c := <-ch:
			n++
			err := c.close()
			if c.err == nil && err == nil {
				return nil
			}

			fmt.Println(err, c.err)

			if n >= len(mxr)+1 {
				return fmt.Errorf("The email `%s` is invalid", email)
			}
		}
	}
}

type errorCloseClosure struct {
	err   error
	close func() error
}

func newErrorCloseClosure(err error, con net.Conn) errorCloseClosure {
	var close func() error
	if con != nil {
		close = con.Close
	} else {
		close = func() error {
			return err
		}
	}

	return errorCloseClosure{
		err,
		close,
	}
}
func smtpVerify(ch chan errorCloseClosure, email, mx string) {
	con, err := net.Dial("tcp", net.JoinHostPort(mx, strconv.Itoa(25)))

	if err != nil {
		ch <- newErrorCloseClosure(err, con)
		return
	}

	stream := make([]byte, 998)
	status := stream[:3]

	fmt.Fprintf(con, "HELO %s\r\n", hostname)
	bufio.NewReader(con).Read(stream)

	if string(status) != "220" {
		ch <- newErrorCloseClosure(errors.New("Service not ready"), con)
		return
	}

	fmt.Fprintf(con, "MAIL FROM: <%s>\r\n", "example@example.org")
	bufio.NewReader(con).Read(stream)

	if string(status) != "250" {
		ch <- newErrorCloseClosure(errors.New(""), con)
		return
	}

	fmt.Fprintf(con, "RCPT TO: <%s>\r\n", email)
	bufio.NewReader(con).Read(stream)

	if string(status) != "250" {
		ch <- newErrorCloseClosure(errors.New(""), con)
		return
	}

	fmt.Fprintf(con, "RSET\r\n")
	bufio.NewReader(con).Read(stream)

	if string(status) != "250" {
		ch <- newErrorCloseClosure(errors.New(""), con)
		return
	}

	fmt.Fprintf(con, "QUIT\r\n")
	bufio.NewReader(con).Read(stream)

	if string(status) != "221" && string(status) != "250" {
		ch <- newErrorCloseClosure(errors.New(""), con)
		return
	}

	ch <- newErrorCloseClosure(nil, con)
}
