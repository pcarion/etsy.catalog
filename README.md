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
| `ETSY_API_KEY` | Every `auth`, `list`, `get`, and `push` command | Etsy app credentials in `keystring:shared_secret` format |
| `ETSY_ACCESS_TOKEN` | `list`, `get`, and `push` | OAuth access token in `user_id.token` format |
| `ETSY_SHOP_ID` | `list`, `get`, and `push` | Numeric Etsy shop ID |
| `ETSY_REFRESH_TOKEN` | `auth refresh` | OAuth refresh token returned by Etsy |

A typical configured shell looks like this:

```sh
export ETSY_API_KEY='keystring:shared_secret'
export ETSY_ACCESS_TOKEN='user_id.access_token'
export ETSY_REFRESH_TOKEN='user_id.refresh_token'
export ETSY_SHOP_ID='12345678'
```

`ETSY_API_KEY`, `ETSY_ACCESS_TOKEN`, and `ETSY_SHOP_ID` can instead be supplied with the global `--api-key`, `--access-token`, and `--shop-id` flags. `ETSY_REFRESH_TOKEN` can be replaced by the `auth refresh --refresh-token` flag.

The access token needs the `listings_r` scope for `list` and `get`, and `listings_w` for `push`. Do not commit any credentials or tokens.

`ETSY_CODE_VERIFIER` and `ETSY_OAUTH_STATE`, shown later in the authorization flow, are temporary values rather than general CLI configuration. The verifier is passed to `auth exchange --verifier`; the state must be compared with the value returned by Etsy.

## Etsy authentication

Etsy Open API v3 uses OAuth 2.0 Authorization Code flow with PKCE. Before starting, create an app on Etsy's **Your Apps** page and register an HTTPS redirect URI. Etsy supplies a keystring and shared secret; set both as the API key:

```sh
export ETSY_API_KEY='keystring:shared_secret'
```

Generate a PKCE authorization URL. The redirect URI must exactly match one registered for the app:

```sh
./bin/etsy auth url --redirect-uri 'https://example.com/etsy/callback'
```

The default scopes are `listings_r listings_w`. Override them when necessary:

```sh
./bin/etsy auth url \
  --redirect-uri 'https://example.com/etsy/callback' \
  --scopes 'listings_r listings_w'
```

Open the printed URL and approve access. Save the printed `ETSY_CODE_VERIFIER` and `ETSY_OAUTH_STATE`. Etsy redirects to the registered URI with `code` and `state` query parameters. Before continuing, verify that the returned `state` exactly equals the printed `ETSY_OAUTH_STATE`; stop if it does not.

Exchange the code using the same redirect URI and the printed verifier:

```sh
./bin/etsy auth exchange \
  --code 'code-from-the-callback-url' \
  --verifier "$ETSY_CODE_VERIFIER" \
  --redirect-uri 'https://example.com/etsy/callback'
```

The command prints `ETSY_ACCESS_TOKEN`, `ETSY_REFRESH_TOKEN`, and `EXPIRES_IN`. Store the tokens securely and export them in the current shell. Avoid putting them in source-controlled files or shell history. Access tokens normally expire after one hour; refresh tokens have a longer lifetime.

```sh
export ETSY_ACCESS_TOKEN='user_id.access_token'
export ETSY_REFRESH_TOKEN='user_id.refresh_token'
```

When the access token expires, request a new token pair:

```sh
./bin/etsy auth refresh
```

The refresh response can contain a replacement refresh token. Update both environment variables with the newly printed values. The CLI deliberately does not persist tokens to disk or refresh them implicitly, so secret storage remains under the user's control.

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
