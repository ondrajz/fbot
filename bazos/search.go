package bazos

import (
	"fmt"
	"html"
	"net/url"

	"github.com/sirupsen/logrus"
)

type SearchQuery struct {
	Query     string
	Section   *AdSection
	Location  string
	Vicinity  int
	PriceFrom int
	PriceTo   int
}

func (q SearchQuery) Search() ([]Ad, error) {
	u, err := q.toUrl()
	if err != nil {
		return nil, err
	}

	logrus.Debugf("searching: %+v", q)

	return GetAdListings(u.String())
}

func (q SearchQuery) queryValues() url.Values {
	v := make(url.Values)
	v.Set("hledat", html.EscapeString(q.Query))
	if q.Section == nil || q.Section.Section == "" {
		v.Set("rubriky", AnySection)
	} else {
		v.Set("rubriky", q.Section.Section)
	}
	v.Set("hlokalita", q.Location)
	v.Set("humkreis", fmt.Sprint(q.Vicinity))
	if q.PriceFrom >= 0 {
		v.Set("cenaod", fmt.Sprint(q.PriceFrom))
	}
	if q.PriceTo > 0 {
		v.Set("cenado", fmt.Sprint(q.PriceTo))
	}
	return v
}

func (q SearchQuery) toUrl() (*url.URL, error) {
	qvals := q.queryValues().Encode()
	urlPath := DefaultDomain + "/search.php"
	if q.Section != nil && q.Section.Section != "" {
		if q.Section.Category == "" {
			urlPath = fmt.Sprintf("%s.%s/search.php", q.Section.Section, DefaultDomain)
		} else {
			urlPath = fmt.Sprintf("%s.%s/%s/", q.Section.Section, DefaultDomain, q.Section.Category)
		}
	}
	u, err := url.Parse("https://" + urlPath + "?" + qvals)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func Search(query string) ([]Ad, error) {
	/*queryEsc := html.EscapeString(query)
	  domain := DefaultDomain
	  rubriky := AnySection
	  hlokalita := ""
	  vicinity := DefaultVicinity
	  priceFrom := ""
	  priceTo := ""

	  u := fmt.Sprintf("https://%s/search.php?hledat=%s&rubriky=%v&hlokalita=%v&humkreis=%v&cenaod=%v&cenado=%v",
	  	domain, queryEsc, rubriky, hlokalita, vicinity, priceFrom, priceTo)*/

	logrus.Debugf("searching for %q", query)

	search := SearchQuery{
		Query:    query,
		Vicinity: DefaultVicinity,
	}
	/*u, err := search.toUrl()
	  if err != nil {
	  	return nil, err
	  }

	  logrus.Infof("searching %q", query)
	  logrus.Debugf("search URL: %v", u)

	  return GetAdListings(u.String())*/

	return search.Search()
}
