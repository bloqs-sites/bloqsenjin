package helpers

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/bloqs-sites/bloqsenjin/pkg/conf"
)

const (
	DOMAINS_BLACKLIST = iota
	DOMAINS_WHITELIST
	DOMAINS_NIL
)

func getDomainsList() map[string][]string {
	domains := conf.MustGetConfOrDefault[map[string]any](nil, "domains")
	domains_list := make(map[string][]string, len(domains))

	for k, v := range domains {
		domains := make([]string, len(v.([]interface{})))
		for i, v := range v.([]interface{}) {
			domains[i] = v.(string)
		}

		domains_list[k] = domains
	}

	return domains_list
}

func GetDomainsListType() ([]string, int) {
	domains_list := getDomainsList()
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
	domains_list := getDomainsList()
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
				return nil;
			}
		}
		return fmt.Errorf("The domain `%s` is a non whitelisted domain", domain)
	}

	return nil
}

func CheckOriginHeader(h *http.Header, r *http.Request) (uint32, error) {
	uri, err := url.ParseRequestURI(r.Header.Get("Origin"))

	if err != nil {
		return http.StatusForbidden, err
	}

	if err := ValidateDomain(uri.Hostname()); err != nil {
		return http.StatusForbidden, err
	} else {
		if GetDomainsType() == DOMAINS_WHITELIST {
			h.Set("Access-Control-Allow-Origin", uri.String())
			h.Add("Vary", "Origin")
		} else {
			h.Set("Access-Control-Allow-Origin", "*")
		}
	}

	return http.StatusOK, nil
}
