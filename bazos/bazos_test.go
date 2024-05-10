package bazos

import (
	"flag"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

var trace = flag.Bool("debug", false, "Enable trace log level")

func TestMain(m *testing.M) {
	flag.Parse()
	if *trace {
		logrus.SetLevel(logrus.TraceLevel)
	} else if testing.Verbose() {
		logrus.SetLevel(logrus.DebugLevel)
	}
	os.Exit(m.Run())
}

func TestAdListing(t *testing.T) {
	const adUrl = "https://auto.bazos.sk/inzerat/150722248/predam-nosic-bicyklov-na-tazne.php"

	t.Logf("getting ad listing: %v", adUrl)

	adListing, err := GetAd(adUrl)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Ad:\n%+v\n", toYaml(adListing))
}

func TestAdListings(t *testing.T) {
	const adUrl = "https://bazos.sk/search.php?hledat=&rubriky=elektro&hlokalita=84107&humkreis=50&cenaod=&cenado=&Submit=H%C4%BEada%C5%A5"
	// const adUrl = "https://elektro.bazos.sk/pracky/?hledat=&rubriky=elektro&hlokalita=84107&humkreis=50&cenaod=&cenado=&Submit=H%C4%BEada%C5%A5&kitx=ano"

	t.Logf("getting ad listings: %v", adUrl)

	adListings, err := GetAdListings(adUrl)
	if err != nil {
		t.Fatal(err)
	}
	// t.Logf("AdListings:\n%+v\n", toYaml(adListings))

	t.Logf("listing %d ad listings", len(adListings))
	for _, ad := range adListings {
		t.Logf("- %v (%v)\n", ad.Title, ad.Price)
	}
}

func TestSearch(t *testing.T) {
	search := SearchQuery{
		Query:    "electrolux",
		Vicinity: 25,
		Section: &AdSection{
			Category: "pracky",
			Section:  "elektro",
		},
	}
	t.Logf("searching: %v", toYaml(search))

	adListings, err := search.Search()
	if err != nil {
		t.Fatal(err)
	}
	// t.Logf("AdListings:\n%+v\n", toYaml(adListings))

	t.Logf("listing %d ad listings", len(adListings))
	for _, ad := range adListings {
		t.Logf("- %v (%v)\n", ad.Title, ad.Price)
	}
}
