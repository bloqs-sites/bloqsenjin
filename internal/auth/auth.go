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
	"time"

	"github.com/bloqs-sites/bloqsenjin/pkg/auth"
	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
	"github.com/bloqs-sites/bloqsenjin/proto"
	"github.com/golang-jwt/jwt/v4"
)

var (
	email_regex *regexp.Regexp

	hostname string

	domains_list map[string][]string
)

const (
	email_regex_str = "(?:[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*|\"(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21\\x23-\\x5b\\x5d-\\x7f]|\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])*\")@(?:(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?|\\[(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?|[a-z0-9-]*[a-z0-9]:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x21-\\x5a\\x53-\\x7f]|\\[\\x01-\\x09\\x0b\\x0c\\x0e-\\x7f])+)\\])"

	domains_blacklist = iota
	domains_whitelist
	domains_nil
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

    if domains, ok := conf.MustGetConf("domains").(map[string][]string); ok {
        domains_list = domains
    } else {
        domains_list = nil
    }
}

type claims struct {
	auth.Payload
	jwt.RegisteredClaims
}

type Auther struct{}

func (a Auther) GenToken(p auth.Payload) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims{
		p,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "bloqsenjin",
			Subject:   p.Client,
		},
	})

	tokenstr, err := token.SignedString([]byte("secret"))

	if err != nil {
		panic(err)
	}

	return tokenstr
}

func (a Auther) VerifyToken(t string, auths uint) bool {
	token, err := jwt.ParseWithClaims(t, &claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("")
		}

		return []byte("secret"), nil
	}, jwt.WithValidMethods([]string{}))

	if err != nil {
		return false
	}

	if claims, ok := token.Claims.(*claims); ok && token.Valid {
		return (claims.Payload.Permissions & auths) == auths
	} else if errors.Is(err, jwt.ErrTokenMalformed) {
		return false
	} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
		return false
	}

	return false
}

func (a *Auther) SignInBasic(c *proto.Credentials_Basic) error {
	if err := a.verifyEmail(c.Basic.GetEmail()); err != nil {
		return err
	}

	return nil
}

func (a *Auther) verifyEmail(email string) error {
	valid := email_regex.Match([]byte(email))

	if !valid {
		return fmt.Errorf("The email `%s` has an invalid format", email)
	}

	email_domain := strings.Split(email, "@")[1]

	v, t := getDomainsListType()
	switch t {
	case domains_nil:
	case domains_blacklist:
		for _, d := range v {
			if d == email_domain {
				return fmt.Errorf("The email `%s` has a blacklisted domain", email)
			}
		}
	case domains_whitelist:
		for _, d := range v {
			if d == email_domain {
				break
			}
		}
		return fmt.Errorf("The email `%s` has a non whitelisted domain", email)
	}

	mxr, err := net.LookupMX(email_domain)

	if err != nil {
		return err
	}

	sort.SliceStable(mxr, func(i, j int) bool {
		return mxr[i].Pref < mxr[j].Pref
	})

	ch, n := make(chan error, len(mxr)+1), 0

	for _, i := range mxr {
		switch t {
		case domains_blacklist:
			for _, d := range v {
				if d == i.Host {
					return fmt.Errorf("The email `%s` has a blacklisted domain", email)
				}
			}
		}
		go a.smtpVerify(ch, email, i.Host)
	}
	go a.smtpVerify(ch, email, email_domain)

	for {
		select {
		case err := <-ch:
			n++
			if err == nil {
				return nil
			}

			fmt.Println(err)

			if n >= len(mxr)+1 {
				return fmt.Errorf("The email `%s` is invalid", email)
			}
		}
	}
}

func getDomainsListType() ([]string, int) {
    if domains_list == nil {
	    return nil, domains_nil
    }

	if v, ok := domains_list["blacklist"]; ok {
		return v, domains_blacklist
	} else if v, ok := domains_list["whitelist"]; ok {
		return v, domains_whitelist
	}

	return nil, domains_nil
}

func (a *Auther) smtpVerify(ch chan error, email, mx string) {
	con, err := net.Dial("tcp", net.JoinHostPort(mx, strconv.Itoa(25)))

	if err != nil {
		ch <- err
	}

	defer con.Close()
	// TODO: needs to Close when email verified already and stop this function execution

	stream := make([]byte, 998)
	status := stream[:3]

	fmt.Fprintf(con, "HELO %s\r\n", hostname)
	bufio.NewReader(con).Read(stream)

	if string(status) != "220" {
		ch <- errors.New("Service not ready")
	}

	fmt.Fprintf(con, "MAIL FROM: <%s>\r\n", "example@example.org")
	bufio.NewReader(con).Read(stream)

	if string(status) != "250" {
		ch <- errors.New("")
	}

	fmt.Fprintf(con, "RCPT TO: <%s>\r\n", email)
	bufio.NewReader(con).Read(stream)

	if string(status) != "250" {
		ch <- errors.New("")
	}

	fmt.Fprintf(con, "RSET\r\n")
	bufio.NewReader(con).Read(stream)

	if string(status) != "250" {
		ch <- errors.New("")
	}

	fmt.Fprintf(con, "QUIT\r\n")
	bufio.NewReader(con).Read(stream)

	if string(status) != "221" && string(status) != "250" {
		ch <- errors.New("")
	}

	con.Close()

	ch <- nil
}
