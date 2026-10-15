# Provider coverage

Verified 2026-10-15. The catalog contains **64 providers in 11 topics**, with
**63 enabled by default**. This matrix describes the configured archive scope;
it does not claim that a provider entry mirrors every upstream document.

Source URLs and path selections are maintained in
[`configs/providers.yaml`](../../configs/providers.yaml). Fetch quirks and
source-use notes are in the [provider notes](README.md). Native bulk sources are
split when their published format supports it. Raw exports and indexes remain
single artifacts; index links are followed only by the bounded AWS guide
strategy. GitHub entries preserve source Markdown, MDX, reStructuredText, or
schema files rather than converting every artifact to Markdown.

Artifact counts below describe successful cold snapshots on the verification
date, including retained indexes, full exports, and supplemental schemas. Each
count was checked against both sync metadata and files in `latest/`. Counts may
change as upstream documentation changes. Examples are relative to
`<archive_root>/<topic>/<provider>/latest/`.

GitHub source discovery normally uses `GITHUB_TOKEN`; raw file downloads do not
receive that token. Jina entries use only `JINA_API_KEY` for the conversion
service. Native public documentation needs no provider API key. All 63 enabled entries completed a cold archive check. A transient Jina HTTP 503
blocked the first YouTube guide attempt; a targeted retry completed all 42 pages.
A failed configured page prevents publication of an incomplete provider snapshot.

## Frontier Labs

| Provider    | Source                                                        | Configured scope                                                                                                                                         | Artifacts | Example                                   |
| ----------- | ------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- | --------: | ----------------------------------------- |
| `xai`       | [index / dump](https://docs.x.ai/llms-full.txt)               | Published dump + curated API, Grok Bot and integration pages                                                                                             |       207 | `build/cli/headless-scripting.md`         |
| `anthropic` | [index / dump](https://platform.claude.com/llms-full.txt)     | Published Claude documentation dump + curated tool/cache pages                                                                                           |       780 | `en/about-claude/additional-resources.md` |
| `openai`    | [API export](https://developers.openai.com/api/llms-full.txt) | API export as one raw document + 4 curated pages; [OpenAPI](https://raw.githubusercontent.com/openai/openai-openapi/refs/heads/manual_spec/openapi.yaml) |         6 | `docs/models.md`                          |

## Inference Hosts

| Provider      | Source                                                           | Configured scope                                                                                                                                  | Artifacts | Example                                                                              |
| ------------- | ---------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- | --------: | ------------------------------------------------------------------------------------ |
| `deepinfra`   | [index / dump](https://docs.deepinfra.com/llms-full.txt)         | Published native dump + 4 curated pages; [schema](https://api.deepinfra.com/openapi.json)                                                         |       225 | `account/authentication.md`                                                          |
| `fireworks`   | [index / dump](https://docs.fireworks.ai/llms-full.txt)          | Published native dump + 9 curated pages                                                                                                           |       392 | `accounts/data-residency.md`                                                         |
| `together`    | [index / dump](https://docs.together.ai/llms-full.txt)           | Published native dump + 6 curated pages                                                                                                           |       415 | `account-management.md`                                                              |
| `groq`        | [index / dump](https://console.groq.com/llms-full.txt)           | Published native dump + 5 curated pages                                                                                                           |       340 | `agno.md`                                                                            |
| `cerebras`    | [index / dump](https://inference-docs.cerebras.ai/llms-full.txt) | Published native dump + 4 curated pages; [schema](https://inference-docs.cerebras.ai/api-reference/openapi.yaml)                                  |       119 | `api-reference/authentication.md`                                                    |
| `sambanova`   | [index / dump](https://docs.sambanova.ai/docs/llms-full.txt)     | Published native dump; [schema](https://raw.githubusercontent.com/sambanova/sambanova-inference-api-spec/refs/heads/main/openapi.documented.json) |       191 | `api-reference/audio/transcribes-audio-into-the-input-language.md`                   |
| `hyperbolic`  | [index / dump](https://www.hyperbolic.ai/docs/llms-full.txt)     | Published native dump + 4 curated pages; [schema](https://www.hyperbolic.ai/docs/api-reference/openapi.json)                                      |       100 | `api-reference/api-keys/list-all-api-keys-for-the-current-user-or-organization-1.md` |
| `featherless` | [index / dump](https://featherless.ai/llms.txt)                  | Index only; linked HTML documentation is not followed                                                                                             |         1 | `llms.txt`                                                                           |
| `novita`      | [index / dump](https://docs.novita.ai/llms-full.txt)             | Published native dump + 3 curated pages                                                                                                           |       433 | `api-reference/api-reference-overview.md`                                            |
| `typesafe`    | [index / dump](https://docs.typesafe.ai/llms-full.txt)           | Published native dump + 9 curated pages; [schema](https://api.typesafe.ai/openapi.json)                                                           |       131 | `agent-skill.md`                                                                     |

## Python Libraries

| Provider   | Source                                                         | Configured scope                                  | Artifacts | Example                    |
| ---------- | -------------------------------------------------------------- | ------------------------------------------------- | --------: | -------------------------- |
| `pydantic` | [index / dump](https://docs.pydantic.dev/latest/llms-full.txt) | Raw full export + 5 curated concept/example pages |         6 | `concepts/config/index.md` |

## Cloud Infrastructure

| Provider                | Source                                                                                   | Configured scope                                              | Artifacts | Example                                          |
| ----------------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------- | --------: | ------------------------------------------------ |
| `aws-glue-dg`           | [index / dump](https://docs.aws.amazon.com/llms.txt)                                     | Guide index + 3 curated developer-guide pages                 |         4 | `set-up-iam.md`                                  |
| `aws-bedrock-userguide` | [index / dump](https://docs.aws.amazon.com/llms.txt)                                     | Guide index + 3 curated user-guide pages                      |         4 | `getting-started.md`                             |
| `aws-bedrock-apiref`    | [index / dump](https://docs.aws.amazon.com/llms.txt)                                     | Guide index + 2 curated API-reference pages                   |         3 | `API_GetFoundationModel.md`                      |
| `digitalocean-api`      | [index / dump](https://docs.digitalocean.com/llms-full-reference-api.txt)                | Area full-text export + sections within configured URL prefix |        81 | `llms-full-reference-api.txt`                    |
| `digitalocean-doctl`    | [index / dump](https://docs.digitalocean.com/llms-full-reference-doctl.txt)              | Area full-text export + sections within configured URL prefix |       726 | `llms-full-reference-doctl.txt`                  |
| `digitalocean-k8s`      | [index / dump](https://docs.digitalocean.com/llms-full-products-kubernetes.txt)          | Area full-text export + sections within configured URL prefix |       426 | `llms-full-products-kubernetes.txt`              |
| `digitalocean-dbs`      | [index / dump](https://docs.digitalocean.com/llms-full-products-databases.txt)           | Area full-text export + sections within configured URL prefix |       240 | `llms-full-products-databases.txt`               |
| `digitalocean-droplets` | [index / dump](https://docs.digitalocean.com/llms-full-products-droplets.txt)            | Area full-text export + sections within configured URL prefix |        67 | `llms-full-products-droplets.txt`                |
| `digitalocean-spaces`   | [index / dump](https://docs.digitalocean.com/llms-full-products-spaces.txt)              | Area full-text export + sections within configured URL prefix |        41 | `llms-full-products-spaces.txt`                  |
| `cloudflare-workers`    | [index / dump](https://developers.cloudflare.com/workers/llms-full.txt)                  | Product full-text dump split by page frontmatter              |       420 | `workers/authorization/durable-objects/index.md` |
| `cloudflare-pages`      | [index / dump](https://developers.cloudflare.com/pages/llms-full.txt)                    | Product full-text dump split by page frontmatter              |       115 | `pages/configuration/api/index.md`               |
| `cloudflare-r2`         | [index / dump](https://developers.cloudflare.com/r2/llms-full.txt)                       | Product full-text dump split by page frontmatter              |        79 | `r2/api/error-codes/index.md`                    |
| `cloudflare-kv`         | [index / dump](https://developers.cloudflare.com/kv/llms-full.txt)                       | Product full-text dump split by page frontmatter              |        28 | `kv.md`                                          |
| `hetzner-cloud-api`     | [GitHub source](https://github.com/MaximilianKoestler/hcloud-openapi/tree/HEAD/openapi/) | Community-maintained hcloud OpenAPI JSON only                 |         1 | `hcloud.json`                                    |
| `hetzner-cloud`         | [pages](https://docs.hetzner.com/cloud)                                                  | 7 curated HTML pages converted to Markdown                    |         7 | `api/getting-started/using-api.md`               |
| `hetzner-networking`    | [pages](https://docs.hetzner.com/networking)                                             | 5 curated HTML pages converted to Markdown                    |         5 | `dns/overview.md`                                |

## Data Platform

| Provider | Source                                                                            | Configured scope                    | Artifacts | Example    |
| -------- | --------------------------------------------------------------------------------- | ----------------------------------- | --------: | ---------- |
| `trino`  | [GitHub source](https://github.com/trinodb/trino/tree/HEAD/docs/src/main/sphinx/) | Markdown under docs/src/main/sphinx |       648 | `admin.md` |

## Container Platform

| Provider             | Source                                                                                              | Configured scope                           | Artifacts | Example          |
| -------------------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------ | --------: | ---------------- |
| `kubernetes-kubectl` | [GitHub source](https://github.com/kubernetes/website/tree/HEAD/content/en/docs/reference/kubectl/) | English kubectl reference Markdown subtree |       120 | `conventions.md` |

## Collaboration Platforms

| Provider               | Source                                                                                                              | Configured scope                                          | Artifacts | Example                               |
| ---------------------- | ------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------- | --------: | ------------------------------------- |
| `mattermost-api`       | [GitHub source](https://github.com/mattermost/mattermost/tree/HEAD/api/v4/source/)                                  | API v4 source YAML fragments; not rendered narrative docs |        59 | `access_control.yaml`                 |
| `mattermost-integrate` | [GitHub source](https://github.com/mattermost/mattermost-developer-documentation/tree/HEAD/site/content/integrate/) | Integration documentation Markdown subtree                |        57 | `apps/authentication/oauth2/index.md` |

## Self-Hosted Suite

| Provider          | Source                                                                                    | Configured scope                              | Artifacts | Example                      |
| ----------------- | ----------------------------------------------------------------------------------------- | --------------------------------------------- | --------: | ---------------------------- |
| `nextcloud-admin` | [GitHub source](https://github.com/nextcloud/documentation/tree/HEAD/admin_manual/)       | Administration manual in reStructuredText     |       194 | `ai/ai_as_a_service.rst`     |
| `stalwart`        | [GitHub source](https://github.com/stalwartlabs/website/tree/HEAD/src/content/docs/docs/) | Published docs source subtree, Markdown + MDX |       489 | `auth/authentication/2fa.md` |

## SpaceX & Starlink Data APIs

| Provider                    | Source                                                                                  | Configured scope                            | Artifacts | Example                                  |
| --------------------------- | --------------------------------------------------------------------------------------- | ------------------------------------------- | --------: | ---------------------------------------- |
| `spacex-api-guides`         | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/)               | README + 4 curated project guides           |         5 | `README.md`                              |
| `spacex-launches-v4`        | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/launches/v4/)   | Resource/version reference Markdown subtree |         8 | `all.md`                                 |
| `spacex-launches-v5`        | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/launches/v5/)   | Resource/version reference Markdown subtree |         9 | `README.md`                              |
| `spacex-rockets-v4`         | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/rockets/v4/)    | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-capsules-v4`        | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/capsules/v4/)   | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-cores-v4`           | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/cores/v4/)      | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-crew-v4`            | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/crew/v4/)       | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-dragons-v4`         | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/dragons/v4/)    | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-history-v4`         | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/history/v4/)    | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-company-v4`         | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/company/v4/)    | Resource/version reference Markdown subtree |         2 | `all.md`                                 |
| `spacex-landpads-v4`        | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/landpads/v4/)   | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-launchpads-v4`      | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/launchpads/v4/) | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-payloads-v4`        | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/payloads/v4/)   | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-roadster-v4`        | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/roadster/v4/)   | Resource/version reference Markdown subtree |         3 | `get.md`                                 |
| `spacex-ships-v4`           | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/ships/v4/)      | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `spacex-starlink-v4`        | [GitHub source](https://github.com/r-spacex/SpaceX-API/tree/master/docs/starlink/v4/)   | Resource/version reference Markdown subtree |         4 | `all.md`                                 |
| `starlink-api-v2-guides`    | [index / dump](https://starlink.readme.io/llms.txt)                                     | Index + 22 curated guide pages              |        23 | `60-day-deprecation-policy.md`           |
| `starlink-api-v2-reference` | [index / dump](https://starlink.readme.io/llms.txt)                                     | Index + 67 curated REST reference pages     |        68 | `delete_public-v2-contacts-subjectid.md` |
| `starlink-api-v2-status`    | [pages](https://starlink.readme.io)                                                     | 6 curated service/status guide pages        |         6 | `docs/location-in-aviation.md`           |

## Social Platform APIs

| Provider                     | Source                                                      | Configured scope                                                                                                          | Artifacts | Example                                        |
| ---------------------------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------- | --------: | ---------------------------------------------- |
| `xdev-platform-guides`       | [index / dump](https://docs.x.com/llms.txt)                 | Index/agent files + 15 curated platform pages; [schema](https://docs.x.com/openapi.json)                                  |        17 | `AGENT.md`                                     |
| `xdev-api-messaging`         | [index / dump](https://docs.x.com/x-api/llms.txt)           | Index + 65 curated messaging/account-activity pages                                                                       |        66 | `x-api/account-activity/create-replay-job.md`  |
| `xdev-enterprise-gnip`       | [index / dump](https://docs.x.com/enterprise-api/llms.txt)  | Index + 14 curated enterprise/GNIP pages                                                                                  |        15 | `x-api/enterprise-gnip-2.0/enterprise-gnip.md` |
| `xdev-api-v2`                | [index / dump](https://docs.x.com/x-api/llms.txt)           | Index + 302 curated X API pages                                                                                           |       303 | `x-api/articles/create-draft-article.md`       |
| `xdev-xdk-python`            | [index / dump](https://docs.x.com/xdks/python/llms.txt)     | Index + 68 curated Python SDK pages                                                                                       |        69 | `xdks/python/authentication.md`                |
| `xdev-xdk-typescript`        | [index / dump](https://docs.x.com/xdks/typescript/llms.txt) | Disabled by default; 105 configured SDK paths; not archive-verified                                                       |  disabled | —                                              |
| `youtube-data-api-guides`    | [pages](https://developers.google.com)                      | 42 curated HTML guide pages                                                                                               |        42 | `youtube/v3/code_samples.md`                   |
| `youtube-data-api-reference` | [pages](https://developers.google.com)                      | 76 curated HTML references + Google Discovery schema; [schema](https://youtube.googleapis.com/$discovery/rest?version=v3) |        77 | `openapi.yaml`                                 |

## Design Platform APIs

| Provider    | Source                                                                     | Configured scope                                            | Artifacts | Example        |
| ----------- | -------------------------------------------------------------------------- | ----------------------------------------------------------- | --------: | -------------- |
| `figma-api` | [GitHub source](https://github.com/figma/rest-api-spec/tree/HEAD/openapi/) | Official OpenAPI YAML only; narrative site is outside scope |         1 | `openapi.yaml` |

## Archive interpretation

- AWS entries fetch only configured pages present in exactly one selected guide
  index. They do not crawl an entire service or follow every index link.
- Featherless currently retains an index. Its successful one-file sync does not
  establish documentation-body coverage.
- Figma and Hetzner Cloud API are schema-only entries; Mattermost API retains
  YAML source fragments. YouTube supplies a Google Discovery document, which is
  not an OpenAPI specification despite sharing the generic schema-fetch setting.
- The disabled TypeScript SDK entry remains available for explicit selection;
  the counts above do not imply a successful full archive check for it.
- Archive paths derived from published page URLs can differ from older
  title-derived paths. New snapshots preserve directory structure and reject
  distinct sources that collide at a destination; historical date snapshots
  remain available.
