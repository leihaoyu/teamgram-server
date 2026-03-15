package webpage

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	FetchTimeout = 5 * time.Second
	MaxBodyBytes = 512 * 1024 // 512KB max HTML to parse
)

var HttpClient = &http.Client{
	Timeout: FetchTimeout,
	Transport: &http.Transport{
		DialContext:        (&net.Dialer{Timeout: 3 * time.Second}).DialContext,
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// OGMeta holds Open Graph metadata extracted from an HTML page.
type OGMeta struct {
	Title       string
	Description string
	SiteName    string
	Type        string
	Image       string
}

// NormalizeURL ensures the URL has a scheme and is valid.
func NormalizeURL(rawURL string) (string, *url.URL, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", nil, err
	}
	return rawURL, parsed, nil
}

// IsPrivateHost blocks SSRF by rejecting private/loopback hostnames.
func IsPrivateHost(host string) bool {
	if host == "localhost" || host == "" {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false // domain name, allow
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()
}

// Fetch downloads the URL and extracts OG meta tags.
func Fetch(rawURL string) (*OGMeta, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "TelegramBot (like TwitterBot)")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")

	resp, err := HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml") {
		return nil, fmt.Errorf("not HTML: %s", ct)
	}

	return ParseOGMeta(io.LimitReader(resp.Body, MaxBodyBytes))
}

// ParseOGMeta parses HTML from a reader and extracts OG meta tags.
func ParseOGMeta(r io.Reader) (*OGMeta, error) {
	tokenizer := html.NewTokenizer(r)
	og := &OGMeta{}
	inTitle := false
	titleDone := false

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return og, nil // partial parse is fine, EOF is normal

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()
			tagName := string(tn)

			if tagName == "title" && !titleDone {
				inTitle = true
				continue
			}

			if tagName == "meta" && hasAttr {
				attrs := readAttrs(tokenizer)
				prop := attrs["property"]
				name := attrs["name"]
				content := attrs["content"]

				switch {
				case prop == "og:title" && content != "":
					og.Title = content
				case prop == "og:description" && content != "":
					og.Description = content
				case prop == "og:site_name" && content != "":
					og.SiteName = content
				case prop == "og:type" && content != "":
					og.Type = content
				case prop == "og:image" && content != "":
					og.Image = content
				case name == "description" && content != "" && og.Description == "":
					og.Description = content
				}
			}

			// Stop at <body> — meta tags are in <head>
			if tagName == "body" {
				return og, nil
			}

		case html.TextToken:
			if inTitle {
				text := strings.TrimSpace(string(tokenizer.Text()))
				if text != "" && og.Title == "" {
					og.Title = text
				}
				titleDone = true
				inTitle = false
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			if string(tn) == "title" {
				inTitle = false
				titleDone = true
			}
			if string(tn) == "head" {
				return og, nil
			}
		}
	}
}

func readAttrs(z *html.Tokenizer) map[string]string {
	attrs := make(map[string]string)
	for {
		key, val, more := z.TagAttr()
		k := string(key)
		if k != "" {
			attrs[k] = string(val)
		}
		if !more {
			break
		}
	}
	return attrs
}
