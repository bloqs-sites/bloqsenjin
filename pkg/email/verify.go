package email

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bloqs-sites/bloqsenjin/internal/helpers"
)

func VerifyEmail(ctx context.Context, email string) error {
	valid := email_regex.MatchString(email)

	if !valid {
		return &InvalidEmailError{email, "invalid format", http.StatusUnprocessableEntity}
	}

	email_domain := strings.Split(email, "@")[1]

	if err := helpers.ValidateDomain(email_domain); err != nil {
		return &InvalidEmailError{email, err.Error(), http.StatusUnprocessableEntity}
	}

	mxr, err := net.DefaultResolver.LookupMX(ctx, email_domain)
	if err != nil {
		return &ServerError{err}
	}

	sort.SliceStable(mxr, func(i, j int) bool {
		return mxr[i].Pref < mxr[j].Pref
	})

	var (
		wg   sync.WaitGroup
		once sync.Once
		ch   = make(chan struct{}, 1)
	)

	v, t := helpers.GetDomainsListType()
	for _, i := range mxr {
		switch t {
		case helpers.DOMAINS_BLACKLIST:
			for _, d := range v {
				if d == i.Host {
					return &InvalidEmailError{email, fmt.Sprintf("uses the blacklisted domain `%s` for a MX record", i.Host), http.StatusUnprocessableEntity}
				}
			}
		}
		wg.Add(1)
		go smtpVerify(email, i.Host, &wg, ch, &once)
	}
	//wg.Add(1)
	//go smtpVerify(email, email_domain, &wg, ch, &once)

	done, wait := false, make(chan struct{}, 1)
	for {
		select {
		case <-ch:
			return nil
		case <-wait:
			return &InvalidEmailError{email, "could not be validated. it might not exist. try other email", http.StatusUnprocessableEntity}
		default:
			if done {
				continue
			}

			done = true

			go func() {
				wg.Wait()

				close(wait)
			}()
		}
	}
}

func smtpVerify(email, mx string, wg *sync.WaitGroup, ch chan struct{}, once *sync.Once) {
	defer wg.Done()

	done := false

	for {
		select {
		case <-ch:
			return
		default:
			if done {
				continue
			}

			done = true

			go func() {
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

				once.Do(func() {
					close(ch)
				})
			}()
		}
	}
}
