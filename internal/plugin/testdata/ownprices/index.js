// A plugin whose vendor bills its own tariff: its config hook says so with
// ownPrices (magpie's own field) on one provider and not on the other.
const provider = (cfg, id, extra) => {
  cfg.provider ??= {}
  cfg.provider[id] = { name: id, npm: "@ai-sdk/openai-compatible", api: "https://" + id + ".invalid/v1", models: {}, ...extra }
}
const auth = (id) => ({ provider: id, methods: [{ type: "api", label: "API key" }] })

export const Tariff = async () => ({ config: async (cfg) => provider(cfg, "tariff", { ownPrices: true }), auth: auth("tariff") })
export const Plain = async () => ({ config: async (cfg) => provider(cfg, "plain", { ownPrices: "yes" }), auth: auth("plain") })
