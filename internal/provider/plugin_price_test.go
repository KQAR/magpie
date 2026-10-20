package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/settings"
)

// gatewayPlugin is a plugin in front of a gateway that bills its own
// tariff, signed in: it serves gpt-5.5, which models.dev prices under
// openai at $5/$30, at $1/$2, and lists one model at 0 and one at no cost.
// own is whether its config hook set ownPrices.
func gatewayPlugin(t *testing.T, own bool) Provider {
	t.Helper()
	claudeHome(t)
	t.Setenv("MAGPIE_BUN", "")
	os.MkdirAll(filepath.Dir(catalog.CachePath()), 0o755)
	os.WriteFile(catalog.CachePath(), []byte(`{
	  "openai":{"id":"openai","models":{
	    "gpt-5.5":{"id":"gpt-5.5","cost":{"input":5,"output":30,"cache_read":0.5}},
	    "gpt-5.4":{"id":"gpt-5.4","cost":{"input":2.5,"output":15,"cache_read":0.25}},
	    "gpt-5.2":{"id":"gpt-5.2","cost":{"input":1.75,"output":14,"cache_read":0.175}}}}}`), 0o644)
	catalog.Reset()
	t.Cleanup(catalog.Reset)
	spec := "opencode-gateway-auth"
	write := func(name string, v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		os.MkdirAll(settings.Dir(), 0o700)
		if err := os.WriteFile(filepath.Join(settings.Dir(), name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("plugins.json", map[string]any{"plugins": []map[string]any{{"spec": spec}}})
	write("plugin-auth.json", map[string]map[string]any{"gw": {"type": "api", "key": "sk-gw-1234"}})
	type cost = plugin.Cost
	priced := cost{Input: 1, Output: 2}
	priced.Cache.Read, priced.Cache.Write = 0.1, 1
	plugin.UseCached([]plugin.Provider{{ID: "gw", Spec: spec, Name: "Gateway", OwnPrices: own,
		Methods:  []plugin.Method{{Type: "api", Label: "API key"}},
		Accounts: []plugin.Account{{Key: "gw", Type: "api"}},
		Models: []plugin.Model{
			{ID: "gpt-5.5", Name: "GPT 5.5", Cost: &priced},
			{ID: "gpt-5.4", Name: "GPT 5.4", Cost: &cost{}},
			{ID: "gpt-5.2", Name: "GPT 5.2"},
		}}})
	t.Cleanup(func() { plugin.UseCached(nil) })
	p, err := Find("gw")
	if err != nil {
		t.Fatal(err)
	}
	return *p
}

// A plugin that says its costs are its vendor's own (ownPrices) is priced
// at them over models.dev's for the same id; a model it gives no cost, or
// 0, keeps models.dev's; the user's own price still goes first. A plugin
// that doesn't say so is priced as before: models.dev's wherever it has one.
func TestPluginOwnPricesGoFirst(t *testing.T) {
	own := catalog.Price{Input: 1, Output: 2, CacheRead: 0.1, CacheWrite: 1}
	maker := map[string]catalog.Price{
		"gpt-5.5": {Input: 5, Output: 30, CacheRead: 0.5},
		"gpt-5.4": {Input: 2.5, Output: 15, CacheRead: 0.25},
		"gpt-5.2": {Input: 1.75, Output: 14, CacheRead: 0.175},
	}

	p := gatewayPlugin(t, true)
	if pr, ok := p.ListPrice("gpt-5.5"); !ok || !pr.Same(own) {
		t.Errorf("ownPrices: ListPrice(gpt-5.5) = %+v %v, want the plugin's %+v", pr, ok, own)
	}
	for _, id := range []string{"gpt-5.4", "gpt-5.2"} {
		if pr, ok := p.ListPrice(id); !ok || !pr.Same(maker[id]) {
			t.Errorf("ownPrices: ListPrice(%s) with no cost of the plugin's = %+v %v, want models.dev's %+v", id, pr, ok, maker[id])
		}
	}
	in, out, read, write := 3.0, 4.0, 0.3, 3.0
	st := settings.Settings{ModelPrices: map[string]settings.ModelPrice{"gw/gpt-5.5": {Input: &in, Output: &out, CacheRead: &read, CacheWrite: &write}}}
	if pr, ok := EffectivePriceIn(st, "gw", "gpt-5.5"); !ok || pr.Input != 3 || pr.Output != 4 {
		t.Errorf("ownPrices: the user's price = %+v %v, want $3/$4", pr, ok)
	}

	p = gatewayPlugin(t, false)
	for id, want := range maker {
		if pr, ok := p.ListPrice(id); !ok || !pr.Same(want) {
			t.Errorf("no ownPrices: ListPrice(%s) = %+v %v, want models.dev's %+v", id, pr, ok, want)
		}
	}
}
