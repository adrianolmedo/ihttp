
# iHTTP

A simple and lightweight HTTP Client, fully using the Golang standard library net/http package as a CLI tool, 
inspired in [HTTPie](https://github.com/httpie/httpie).

## TO-DO

This proposal is still a work in progress, it is not ready to be used properly for testing REST APIs.

[-] Enable all other separators by default.

[-] Add command-line colors with [gookit/color](https://github.com/gookit/color).

## Content

* [Usage](#usage)
  * [HTTP Method](#http-method)
  * [Query string params](#query-string-params)
  * [HTTP Headers](#http-headers)
  * [Shortcut for localhost](#shortcut-for-localhost)
  * [Scheme](#scheme)
* [Roadmap](#roadmap)

## Usage

The name of the HTTP Method comes right before the URL argument:

```bash
$ http [flags] [METHOD] URL [ITEMS ...]
```

Execute `http -help` for more detailed information.

### HTTP Method

The METHOD command is optional, by default sends `GET` request without using it:

```bash
$ http httpbingo.org/get
```

Both ways will send a `GET` request:

```bash
$ http GET httpbingo.org/get
```

Make `GET` requests containing **a body** with `foo=bar` notation:

```bash
$ http GET httpbingo.org/get hello=world
```

---

Send a `POST` request automatically specifying **a body**:

```bash
$ http httpbingo.org/post hello=world
```

So use `POST` for requests with **body**:

```bash
$ http POST httpbingo.org/post hello=world
```

### Query string params

It admit the `param==value` notation for add parameters to the URL

```bash
$ http https://api.github.com/search/repositories q==go per_page==1
```

Any special characters are automatically escaped from the URL.

Unlike the original parameters in the URL, which will not be altered either by iHTTP.

### HTTP Headers

Set custom headers with the `Header:Value` notation:

```bash
$ http -v httpbingo.org/headers User-Agent:Bacon/1.0 'Cookie:valued-visitor=yes;foo=bar' \
    X-Foo:Bar Referer:https://httpbingo.org/
```

Pass `-v` flag for see the HTTP Request in the output:

```bash
GET /headers HTTP/1.1
Accept-Encoding: gzip
Cookie: valued-visitor=yes;foo=bar
Host: httpbingo.org
Referer: https://httpbingo.org/
User-Agent: Bacon/1.0
X-Foo: Bar
```

**NOTE:** iHTTP has no HTTP Headers by default, only those determined by the Go `net/http` stdlib.

### Shortcut for localhost

Supports curl-like shorthand for localhost:

```bash
$ http :
```

The URL is processed as `http://localhost`.

If the port is omitted, the port `80` is assumed.

```bash
$ http -v :
GET / HTTP/1.1
Host: localhost
```

Another example:

```bash
$ http :3000/bar
```

The URL is processed as `http://localhost:3000`:

```bash
$ http -v :3000/bar
GET /bar HTTP/1.1
Host: localhost:3000
```

### Scheme

You can install the iHTTP executable as `https`:

```bash
$ https example.org 
```

Which the default scheme will be `https://`, it will make a request to `https://example.org`.

Also you can change the scheme value with `-scheme`:

```bash
$ http -scheme https example.org
```

Again this will make a request to `https://example.org`.

## Roadmap

- API for add new HTTP Methods and separators.
- Plugin system for extend functionalities.
