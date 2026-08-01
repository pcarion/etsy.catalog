# etsy.catalog

`etsy` is a Cobra-based Go CLI that keeps Etsy listings as editable YAML catalogs.

## Build and configure

```sh
make build
```

## Environment variables

The CLI uses these environment variables:

| Variable | Required for | Value |
| --- | --- | --- |
| `ETSY_API_KEY` | Every `auth`, `shop`, `list`, `get`, and `push` command | Etsy app credentials in `keystring:shared_secret` format |
| `ETSY_ACCESS_TOKEN` | `shop`, `list`, `get`, and `push` | OAuth access token in `user_id.token` format |
| `ETSY_SHOP_ID` | `list`, `get`, and `push` | Numeric Etsy shop ID |
| `ETSY_REFRESH_TOKEN` | `auth refresh` | OAuth refresh token returned by Etsy |
| `ETSY_AUTH_REDIRECT_URL` | `auth login` and `auth url` | HTTPS callback URL registered for the Etsy app |
| `ETSY_AUTH_RELAY_URL` | `auth login` | Base URL of the CLI OAuth callback relay |
| `ETSY_AUTH_CALLBACK_TOKEN` | `auth login` | Bearer token protecting the callback relay API |
| `ETSY_CREDENTIALS_FILE` | Optional | Override the platform-specific credential file location |

A typical configured shell looks like this:

```sh
export ETSY_API_KEY='keystring:shared_secret'
export ETSY_ACCESS_TOKEN='user_id.access_token'
export ETSY_REFRESH_TOKEN='user_id.refresh_token'
export ETSY_SHOP_ID='12345678'
export ETSY_AUTH_REDIRECT_URL='https://webhooks.example.com/oauth/cli/callback'
export ETSY_AUTH_RELAY_URL='https://webhooks.example.com'
export ETSY_AUTH_CALLBACK_TOKEN='secret-relay-bearer-token'
```

`ETSY_API_KEY`, `ETSY_ACCESS_TOKEN`, and `ETSY_SHOP_ID` can instead be supplied with the global `--api-key`, `--access-token`, and `--shop-id` flags. `ETSY_REFRESH_TOKEN` can be replaced by `auth refresh --refresh-token`, and `ETSY_AUTH_REDIRECT_URL` by `--redirect-uri` on the authorization commands.

The access token needs the `listings_r` scope for `list` and `get`, and `listings_w` for `push`. Do not commit any credentials or tokens.

Environment variables take precedence over saved credentials. Successful `auth login` and `auth refresh` commands automatically save access and refresh tokens in the operating system's user configuration directory with file mode `0600`. If old token variables remain exported, unset them so they do not override a newer saved login:

```sh
unset ETSY_ACCESS_TOKEN ETSY_REFRESH_TOKEN
```

The default credential location is:

- macOS: `~/Library/Application Support/etsy.catalog/credentials.json`
- Linux: `~/.config/etsy.catalog/credentials.json` (or `$XDG_CONFIG_HOME/etsy.catalog/credentials.json`)
- Windows: the user configuration directory under `etsy.catalog/credentials.json`

Set `ETSY_CREDENTIALS_FILE` to override this location. The credential file contains secrets and must never be committed or shared.

After authentication, retrieve the shop associated with the access token:

```sh
./bin/etsy shop
```

To configure `ETSY_SHOP_ID` directly in a POSIX-compatible shell:

```sh
export ETSY_SHOP_ID="$(./bin/etsy shop --id-only)"
```

## Etsy authentication

Etsy Open API v3 uses OAuth 2.0 Authorization Code flow with PKCE. Before starting, create an app on Etsy's **Your Apps** page and register an HTTPS redirect URI. Etsy supplies a keystring and shared secret, which are configured through `ETSY_API_KEY` as shown above.

### Automated login with a callback relay

Register the dedicated CLI callback URL in the Etsy app, then configure the relay. Do not use the callback service's server-managed `/oauth/callback` route:

```sh
export ETSY_AUTH_REDIRECT_URL='https://webhooks.rememberwherestudio.net/oauth/cli/callback'
export ETSY_AUTH_RELAY_URL='https://webhooks.rememberwherestudio.net'
export ETSY_AUTH_CALLBACK_TOKEN='your-rotated-callback-relay-token'
```

Run:

```sh
./bin/etsy auth login
```

The CLI registers a one-time state, opens Etsy in the browser, waits for the callback, exchanges the authorization code, consumes the relay session, and saves the returned credentials. The only browser interaction is approving access. Use `--no-open` to print the authorization URL in a headless environment.

When the access token expires, request a new token pair:

```sh
./bin/etsy auth refresh
```

The refresh response can contain a replacement refresh token. The CLI saves both new tokens to its credential file. The lower-level `auth url` command remains available for inspecting or troubleshooting generated Etsy authorization URLs, but the supported authorization workflow is `auth login`.

## Commands

```sh
./etsy list
./etsy list --state draft
./etsy get 123456789
./etsy push 123456789 --dry-run
./etsy push 123456789
```

`get` writes `listings/<catalog-id>/listing.yaml`. The YAML retains Etsy's complete listing response under `listing`, inventory under `inventory`, and image/file metadata under `images` and `files`.

`push` updates a catalog with `etsy_listing_id`. Remove that field (or create a new catalog without it) to create a draft. The returned Etsy ID is saved into YAML. Only Etsy-writable fields are sent, so response metadata can remain in the document. Inventory is synchronized through Etsy's inventory endpoint.

To upload assets, place them inside the catalog directory and add a relative `path`; omit `etsy_id` for a new upload:

```yaml
images:
  - path: hero.jpg
    rank: 1
    alt_text: Handmade ceramic bowl
files:
  - path: printable.pdf
```

Existing assets exported by `get` retain their `etsy_id` and are not uploaded again. Push is non-destructive: it adds new local assets but does not delete remote assets absent from YAML.

For a new listing, Etsy requires `quantity`, `title`, `description`, `price`, `who_made`, `when_made`, and `taxonomy_id`. Physical listings also require shipping and readiness profile IDs. If YAML requests `state: active`, the listing is created as a draft, assets are uploaded, then it is activated.

Use `--root` to store catalogs somewhere other than `listings`.
