package email

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
)

func VerifyEmail(ctx context.Context, email string) error {
	valid := email_regex.MatchString(email)

	if !valid {
		return fmt.Errorf("the email `%s` has an invalid format", email)
	}

	email_domain := strings.Split(email, "@")[1]

	if err := helpers.ValidateDomain(email_domain); err != nil {
		return err
	}

	mxr, err := net.DefaultResolver.LookupMX(ctx, email_domain)
	if err != nil {
		return err
	}

	sort.SliceStable(mxr, func(i, j int) bool {
		return mxr[i].Pref < mxr[j].Pref
	})

	var wg sync.WaitGroup
	ch := make(chan struct{}, 1)

	v, t := helpers.GetDomainsListType()
	for _, i := range mxr {
		switch t {
		case helpers.DOMAINS_BLACKLIST:
			for _, d := range v {
				if d == i.Host {
					return fmt.Errorf("the email `%s` uses a blacklisted domain for a MX record", email)
				}
			}
		}
		wg.Add(1)
		go smtpVerify(email, i.Host, &wg, ch)
	}
	wg.Add(1)
	go smtpVerify(email, email_domain, &wg, ch)

	done := false
	for {
		select {
		case <-ch:
			return nil
		default:
			if !done {
				done = true

				wg.Wait()

				return fmt.Errorf("the email `%s` could not be validated. it might not exist or the domain isn't allowed", email)
			}
		}
	}
}

func smtpVerify(email, mx string, wg *sync.WaitGroup, ch chan struct{}) {
	defer wg.Done()

	done := false

	for {
		select {
		case <-ch:
			return
		default:
			if !done {
				done = true

				con, err := net.Dial("tcp", net.JoinHostPort(mx, strconv.Itoa(25)))

				if err != nil {
					return
				}

				defer con.Close()

				stream := make([]byte, 998)
				status := stream[:3]

				fmt.Fprintf(con, "HELO %s\r\n", hostname)
				bufio.NewReader(con).Read(stream)

				if string(status) != "220" {
					return
				}

				fmt.Fprintf(con, "MAIL FROM: <%s>\r\n", "example@example.org")
				bufio.NewReader(con).Read(stream)

				if string(status) != "250" {
					return
				}

				fmt.Fprintf(con, "RCPT TO: <%s>\r\n", email)
				bufio.NewReader(con).Read(stream)

				if string(status) != "250" {
					return
				}

				fmt.Fprintf(con, "RSET\r\n")
				bufio.NewReader(con).Read(stream)

				if string(status) != "250" {
					return
				}

				fmt.Fprintf(con, "QUIT\r\n")
				bufio.NewReader(con).Read(stream)

				if string(status) != "221" && string(status) != "250" {
					return
				}

				close(ch)
			}
		}
	}
}
