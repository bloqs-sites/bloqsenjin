package helpers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
)

var (
	domains_list = conf.MustGetConfOrDefault[map[string][]string](nil, "domains")
)

const (
	DOMAINS_BLACKLIST = iota
	DOMAINS_WHITELIST
	DOMAINS_NIL
)

func GetDomainsListType() ([]string, int) {
	if domains_list == nil {
		return nil, DOMAINS_NIL
	}

	if v, ok := domains_list["blacklist"]; ok {
		return v, DOMAINS_BLACKLIST
	} else if v, ok := domains_list["whitelist"]; ok {
		return v, DOMAINS_WHITELIST
	}

	return nil, DOMAINS_NIL
}

func GetDomainsType() int {
	if domains_list == nil {
		return DOMAINS_NIL
	}

	if _, ok := domains_list["blacklist"]; ok {
		return DOMAINS_BLACKLIST
	} else if _, ok := domains_list["whitelist"]; ok {
		return DOMAINS_WHITELIST
	}

	return DOMAINS_NIL
}

func ValidateDomain(domain string) error {
	v, t := GetDomainsListType()
	switch t {
	case DOMAINS_BLACKLIST:
		for _, d := range v {
			if d == domain {
				return fmt.Errorf("The domain `%s` is a blacklisted domain", domain)
			}
		}
	case DOMAINS_WHITELIST:
		for _, d := range v {
			if d == domain {
				break
			}
		}
		return fmt.Errorf("The domain `%s` is a non whitelisted domain", domain)
	}

	return nil
}

func CheckOriginHeader(w http.ResponseWriter, r *http.Request) error {
	uri, err := url.ParseRequestURI(r.Header.Get("Origin"))

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	if err := ValidateDomain(uri.Hostname()); err != nil {
		w.WriteHeader(http.StatusForbidden)
		return err
	} else {
		if GetDomainsType() == DOMAINS_WHITELIST {
			w.Header().Set("Access-Control-Allow-Origin", uri.String())
			w.Header().Add("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
	}

	return nil
}
