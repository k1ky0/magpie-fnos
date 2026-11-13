// fnOS unified-gateway reverse proxy for magpie.
// Listens on a Unix socket, strips the gateway prefix, forwards to magpie's
// loopback HTTP page, injects the magpie web cookie (so the page needs no key
// in the URL), re-adds the prefix to redirects, and injects a tiny fetch shim
// because magpie's UI calls absolute /api/... paths.
package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
)

func injectCookie(h http.Header, name, val string) {
	var kept []string
	for _, c := range h.Values("Cookie") {
		for _, part := range strings.Split(c, ";") {
			part = strings.TrimSpace(part)
			if part == "" || strings.HasPrefix(part, name+"=") {
				continue
			}
			kept = append(kept, part)
		}
	}
	kept = append(kept, name+"="+val)
	h.Set("Cookie", strings.Join(kept, "; "))
}

func shim(base string) string {
	return `<script>(function(){var B="` + base + `";` +
		`function fix(u){try{if(typeof u==="string"&&u.charAt(0)==="/"&&u.charAt(1)!=="/"&&u.indexOf(B)!==0){return B+u.slice(1);}}catch(e){}return u;}` +
		`var of=window.fetch;if(of){window.fetch=function(i,o){return of.call(this,fix(i),o);};}` +
		`var oe=window.EventSource;if(oe){var E=function(u,c){return c!==undefined?new oe(fix(u),c):new oe(fix(u));};E.prototype=oe.prototype;window.EventSource=E;}` +
		`var ow=window.WebSocket;if(ow){var W=function(u,p){return p!==undefined?new ow(fix(u),p):new ow(fix(u));};W.prototype=ow.prototype;window.WebSocket=W;}` +
		`})();</script>`
}

func main() {
	socket := flag.String("socket", "", "unix socket path to listen on")
	target := flag.String("target", "http://127.0.0.1:13430", "upstream base URL")
	prefix := flag.String("prefix", "/app/magpie", "gateway path prefix to strip")
	cookieName := flag.String("cookie-name", "magpie_web_13430", "magpie web cookie name to inject")
	cookieValue := flag.String("cookie-value", "", "magpie web key")
	flag.Parse()

	tu, err := url.Parse(*target)
	if err != nil {
		log.Fatalf("bad target: %v", err)
	}
	p := httputil.NewSingleHostReverseProxy(tu)
	base := strings.TrimRight(*prefix, "/") + "/"

	p.Director = func(r *http.Request) {
		origHost := r.Host
		r.URL.Scheme = tu.Scheme
		r.URL.Host = tu.Host
		r.Host = origHost
		// strip gateway prefix
		if *prefix != "" {
			if r.URL.Path == *prefix {
				r.URL.Path = "/"
			} else if strings.HasPrefix(r.URL.Path, *prefix+"/") {
				r.URL.Path = strings.TrimPrefix(r.URL.Path, *prefix)
			}
			r.URL.RawPath = ""
		}
		// let Go's transport negotiate/decompress gzip so we can inject HTML
		r.Header.Del("Accept-Encoding")
		// magpie web cookie, so the page needs no ?k= in the URL
		if *cookieValue != "" {
			injectCookie(r.Header, *cookieName, *cookieValue)
		}
	}

	p.ModifyResponse = func(resp *http.Response) error {
		// keep redirects under the gateway prefix
		if loc := resp.Header.Get("Location"); strings.HasPrefix(loc, "/") && !strings.HasPrefix(loc, "//") {
			resp.Header.Set("Location", *prefix+loc)
		}
		ct := resp.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "text/html") {
			b, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return err
			}
			s := string(b)
			ins := shim(base)
			if i := strings.Index(strings.ToLower(s), "<head>"); i >= 0 {
				j := i + len("<head>")
				s = s[:j] + ins + s[j:]
			} else {
				s = ins + s
			}
			resp.Body = io.NopCloser(strings.NewReader(s))
			resp.ContentLength = int64(len(s))
			resp.Header.Set("Content-Length", strconv.Itoa(len(s)))
			resp.Header.Del("Content-Encoding")
		}
		return nil
	}

	if *socket == "" {
		log.Fatal("-socket is required")
	}
	os.Remove(*socket)
	ln, err := net.Listen("unix", *socket)
	if err != nil {
		log.Fatalf("listen %s: %v", *socket, err)
	}
	// fnOS gateway (root) connects regardless; keep it owner/group-writable like other apps
	_ = os.Chmod(*socket, 0o660)
	log.Printf("magpie gateway proxy: %s -> %s (strip %s)", *socket, *target, *prefix)
	log.Fatal(http.Serve(ln, p))
}
