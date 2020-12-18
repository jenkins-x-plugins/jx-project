package root

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// SameRootDomain returns true if the same last 2 paths of the domain are the same. e.g. *.github.com or *.mygitserver.com
func SameRootDomain(u1 string, u2 string) (bool, error) {
	url1, err := url.Parse(u1)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse URL %s", u1)
	}
	url2, err := url.Parse(u2)
	if err != nil {
		return false, errors.Wrapf(err, "failed to parse URL %s", u2)
	}
	domains1 := strings.Split(url1.Host, ".")
	domains2 := strings.Split(url2.Host, ".")
	if len(domains1) < 2 || len(domains2) < 2 {
		return false, nil
	}
	if domains1[len(domains1)-1] != domains2[len(domains2)-1] {
		return false, nil
	}
	if domains1[len(domains1)-2] != domains2[len(domains2)-2] {
		return false, nil
	}
	return true, nil
}
