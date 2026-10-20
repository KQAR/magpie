package plugin

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// A plugin's config hook that sets ownPrices: true on a provider says its
// costs are its vendor's own tariff; anything else is not said.
func TestPluginOwnPricesPassedOn(t *testing.T) {
	sandbox(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	abs, _ := filepath.Abs("testdata/ownprices")
	if _, err := Add(ctx, abs); err != nil {
		t.Fatal(err)
	}
	ps, err := Providers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Provider{}
	for _, p := range ps {
		by[p.ID] = p
	}
	if !by["tariff"].OwnPrices {
		t.Errorf("tariff's OwnPrices = false, want true: %+v", by["tariff"])
	}
	if by["plain"].OwnPrices {
		t.Error(`plain's OwnPrices = true from ownPrices: "yes", want false`)
	}
}
