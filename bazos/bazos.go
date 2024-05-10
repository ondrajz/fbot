package bazos

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gookit/color"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	DefaultDomain   = "bazos.sk"
	DefaultVicinity = 10
)

const (
	PriceFree   = "Zadarmo"
	PriceInText = "Vtexte"
)

type Ad struct {
	ID          string
	Title       string
	Date        time.Time `json:",omitempty" yaml:",omitempty"`
	Link        string
	Section     *AdSection `json:",omitempty" yaml:",omitempty"`
	Price       float64
	Description string
	Image       string   `json:",omitempty" yaml:",omitempty"`
	Images      []string `json:",omitempty" yaml:",omitempty"`

	Location    string
	PostCode    string
	Email       string `json:",omitempty" yaml:",omitempty"`
	UserName    string `json:",omitempty" yaml:",omitempty"`
	PhoneNumber string `json:",omitempty" yaml:",omitempty"`

	Views int `json:",omitempty" yaml:",omitempty"`
}

func GetAd(u string) (*Ad, error) {
	logrus.Debugf("fetching ad from url: %v", u)

	resp, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ad url: %w", err)
	}
	defer resp.Body.Close()

	adUrl := resp.Request.URL.String()
	if adUrl != u {
		logrus.Debugf("ad URL: %v", adUrl)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	ad, err := parseAdListing(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	ad.Link = adUrl

	return ad, nil
}

// GetAdById fetches an Ad with given id.
func GetAdById(id string) (*Ad, error) {
	if id == "" {
		return nil, fmt.Errorf("invalid id")
	}
	log := logrus.WithField("id", id)

	log.Debugf("fetching ad by id")

	ads, err := Search(id)
	if err != nil {
		return nil, fmt.Errorf("searching id %q error: %w", id, err)
	}
	if len(ads) > 1 {
		log.Warnf("got multiple (%v) search results for id %q", len(ads), id)
	} else if len(ads) == 1 {
		log.Debugf("found ad by id: %v", ads[0].Title)
	} else {
		return nil, fmt.Errorf("ad not found")
	}
	ad := ads[0]
	return &ad, nil
}

func GetAdListings(u string) ([]Ad, error) {
	/*logrus.Debugf("getting ad listings from url: %v", url)

	  response, err := http.GetAdById(url)
	  if err != nil {
	  	fmt.Println("Error fetching URL:", err)
	  	return nil, err
	  }
	  defer response.Body.Close()

	  logrus.Debugf("adListings URL: %v", response.Request.URL)

	  body, err := io.ReadAll(response.Body)
	  if err != nil {
	  	fmt.Println("Error reading response body:", err)
	  	return nil, err
	  }

	  logrus.Tracef("parsing ad listings:\n%s", body)

	  // Parse the ad listings
	  adListingsPage, err := parseAdListingsPage(strings.NewReader(string(body)))
	  if err != nil {
	  	return nil, err
	  }*/
	var (
		adListings     []Ad
		adListingsPage *AdListingsPage
	)
	/*adListingsPage, err := getAdListingsPage(url)
	  if err != nil {
	  	return nil, err
	  }

	  adListings = append(adListings, adListingsPage.AdListings...)*/

	listingUrl, err := url.Parse(u)
	if err != nil {
		return nil, err
	}

	page := 0
	for nextUrl := listingUrl.String(); nextUrl != ""; {
		var err error
		adListingsPage, err = getAdListingsPage(nextUrl)
		if err != nil {
			return nil, err
		}

		adListings = append(adListings, adListingsPage.AdListings...)

		page++
		if page >= 10 {
			break
		}

		if adListingsPage.NextPage != "" {
			next, err := listingUrl.Parse(adListingsPage.NextPage)
			if err != nil {
				return nil, err
			}
			nextUrl = next.String()
		} else {
			nextUrl = ""
		}
	}

	return adListings, nil
}

func getAdListingsPage(u string) (*AdListingsPage, error) {
	logrus.Debugf("fetching ad listings page from url: %v", u)

	response, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer response.Body.Close()

	logrus.Debugf("response status: %v", response.Status)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	logrus.Tracef("parsing ad listing page body:\n%s", color.Gray.Sprintf("%s", body))

	// Parse the ad listings
	adListingsPage, err := parseAdListingsPage(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ad listings page: %w", err)
	}

	logrus.Tracef("ad listings page %d, From: %d, To: %d, Total: %d, Next: %s",
		adListingsPage.Page, adListingsPage.From, adListingsPage.To, adListingsPage.Total, adListingsPage.NextPage)

	return adListingsPage, nil
}

func parseSection(doc *goquery.Document) (*AdSection, error) {
	sections := make([]string, 0)

	drobky := doc.Find(".drobky").Children()
	drobky.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		sections = append(sections, s.Text())
	})
	if len(sections) == 0 {
		return nil, fmt.Errorf("no ad sections found")
	}
	logrus.Debugf("parsed sections: %+v", sections)
	section := &AdSection{
		Category: sections[0],
	}
	if len(sections) > 1 {
		section.Section = sections[1]
	}
	return section, nil
}

func parseAdID(s *goquery.Selection) (string, error) {
	adID := ""
	s.Find(".inzeratynadpis a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			parsedURL, err := url.Parse(href)
			if err == nil {
				pathSegments := strings.Split(parsedURL.Path, "/")
				if len(pathSegments) > 2 {
					adID = pathSegments[2]
				}
			}
		}
	})

	if adID == "" {
		return "", fmt.Errorf("ad ID not found")
	}

	return adID, nil
}

type AdListingsPage struct {
	AdListings []Ad
	Page       int
	From, To   int
	Total      int
	NextPage   string
}

func parseAdListingsInfo(s string) (from, to, total int, err error) {
	if strings.Contains(s, "Hľadaniu nevyhovujú žiadne inzeráty") {
		return 0, 0, 0, fmt.Errorf("no ads found")
	}

	re := regexp.MustCompile(`(\d+)-(\d+)\D+(\d+)`)
	matches := re.FindStringSubmatch(s)
	logrus.Tracef("parsing ad listing info: %q, Matches: %+v", s, matches)
	if len(matches) != 4 {
		return 0, 0, 0, fmt.Errorf("incorrect matches count")
	}
	from, err = strconv.Atoi(matches[1])
	if err != nil {
		return
	}
	to, err = strconv.Atoi(matches[2])
	if err != nil {
		return
	}
	total, err = strconv.Atoi(matches[3])
	return
}

func parseAdListingsPage(r io.Reader) (*AdListingsPage, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var from, to, total, page int

	// parse ad listings info
	doc.Find(".listainzerat.inzeratyflex").Each(func(i int, s *goquery.Selection) {
		s.Find(".inzeratynadpis").Each(func(i int, s *goquery.Selection) {
			var err error
			from, to, total, err = parseAdListingsInfo(s.Text())
			if err != nil {
				logrus.Warnf("failed to parse ad listings info: %v", err)
			}
		})
	})
	// parse current page
	doc.Find(".strankovani .cisla").Each(func(i int, s *goquery.Selection) {
		page, _ = strconv.Atoi(s.Text())
	})
	// parse next page
	nextPage := ""
	doc.Find(".strankovani a").Each(func(i int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "Ďalšia" {
			if href, ok := s.Attr("href"); ok {
				nextPage = href
			}
		}
	})

	// parse ad section
	section, err := parseSection(doc)
	if err != nil {
		logrus.Debugf("failed to parse section: %v", err)
	}

	var listings []Ad
	doc.Find(".inzeraty.inzeratyflex").Each(func(i int, s *goquery.Selection) {
		adId, err := parseAdID(s)
		if err != nil {
			logrus.Warnf("failed to parse ad ID: %v", err)
		}
		title := s.Find(".nadpis a").Text()
		link, _ := s.Find(".nadpis a").Attr("href")
		imageURL, _ := s.Find(".obrazek").Attr("src")
		description := s.Find(".popis").Text()
		dateStr := s.Find(".inzeratynadpis span").First().Text()
		dateStr = strings.Trim(dateStr, " -")
		dateStr = strings.TrimPrefix(dateStr, "TOP")
		dateStr = strings.Trim(dateStr, " -")
		dateStr = strings.TrimSpace(strings.Trim(dateStr, "[]"))
		date, err := time.Parse("2.1. 2006", dateStr)
		if err != nil {
			logrus.Warnf("parsing date error: %v", err)
		}
		loc, _ := s.Find(".inzeratylok").Html()
		location := strings.ReplaceAll(loc, "<br/>", " ")
		views := parseViews(s.Find(".inzeratyview").Text())
		price := parsePrice(s.Find(".inzeratycena b").Text())

		listings = append(listings, Ad{
			ID:          adId,
			Section:     section,
			Title:       title,
			Link:        link,
			Images:      []string{imageURL},
			Description: description,
			Date:        date,
			Price:       price,
			Location:    location,
			Views:       views,
		})
	})

	return &AdListingsPage{
		AdListings: listings,
		Page:       page,
		From:       from,
		To:         to,
		Total:      total,
		NextPage:   nextPage,
	}, nil
}

func parseViews(text string) int {
	viewsText := strings.TrimSpace(strings.TrimSuffix(text, "x"))
	if viewsText == "" {
		return 0
	}
	views, err := strconv.Atoi(viewsText)
	if err != nil {
		logrus.Warnf("parsing views (%q) error: %v", viewsText, err)
	}
	return views
}

func parsePrice(text string) float64 {
	priceText := strings.ReplaceAll(text, " ", "")
	priceText = strings.TrimSpace(strings.TrimSuffix(priceText, "€"))
	if priceText == "" || priceText == PriceFree {
		return 0
	} else if priceText == PriceInText {
		return -1
	}
	price, err := strconv.ParseFloat(priceText, 64)
	if err != nil {
		logrus.Warnf("parsing price (%q) error: %v", priceText, err)
	}
	return price
}

func parseAdListing(r io.Reader) (*Ad, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	section, err := parseSection(doc)
	if err != nil {
		logrus.Debugf("failed to parse section: %v", err)
	}

	title := doc.Find(".inzeratydetnadpis H1").Text()
	dateStr := doc.Find(".inzeratydetnadpis span").Text()
	date := parseDate(dateStr)
	description := doc.Find(".popisdetail").Text()

	userName := doc.Find("table table tr td a").First().Text()
	phoneNumber := "Please manually fetch the phone number."
	location := doc.Find("table table tr td a").Last().Text()

	views := parseViews(doc.Find("table table tr td").Eq(11).Text())
	price := parsePrice(doc.Find("table table tr td b").Last().Text())

	var images []string
	doc.Find(".carousel-cell img").Each(func(i int, s *goquery.Selection) {
		src, _ := s.Attr("data-flickity-lazyload")
		images = append(images, src)
	})

	return &Ad{
		Section:     section,
		Title:       title,
		Date:        date,
		UserName:    userName,
		PhoneNumber: phoneNumber,
		Location:    location,
		Views:       views,
		Price:       price,
		Description: description,
		Images:      images,
	}, nil
}

func parseDate(dateStr string) time.Time {
	dateStr = strings.Trim(dateStr, " -[]")
	layout := "2.1. 2006"
	date, _ := time.Parse(layout, dateStr)
	return date
}

func toYaml(v any) string {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return err.Error()
	}
	return buf.String()
}
