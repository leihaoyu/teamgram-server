package webpage

import (
	"strings"
	"testing"
)

func TestParseOGMeta_FullOG(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
<meta property="og:title" content="Test Page Title" />
<meta property="og:description" content="This is a test description" />
<meta property="og:site_name" content="TestSite" />
<meta property="og:type" content="article" />
<meta property="og:image" content="https://example.com/image.jpg" />
<title>Fallback Title</title>
</head>
<body></body>
</html>`

	og, err := ParseOGMeta(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.Title != "Test Page Title" {
		t.Errorf("Title = %q, want %q", og.Title, "Test Page Title")
	}
	if og.Description != "This is a test description" {
		t.Errorf("Description = %q, want %q", og.Description, "This is a test description")
	}
	if og.SiteName != "TestSite" {
		t.Errorf("SiteName = %q, want %q", og.SiteName, "TestSite")
	}
	if og.Type != "article" {
		t.Errorf("Type = %q, want %q", og.Type, "article")
	}
	if og.Image != "https://example.com/image.jpg" {
		t.Errorf("Image = %q, want %q", og.Image, "https://example.com/image.jpg")
	}
}

func TestParseOGMeta_FallbackTitle(t *testing.T) {
	htmlContent := `<html><head>
<title>My Page Title</title>
<meta name="description" content="Meta description here" />
</head><body></body></html>`

	og, err := ParseOGMeta(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.Title != "My Page Title" {
		t.Errorf("Title = %q, want %q (fallback from <title>)", og.Title, "My Page Title")
	}
	if og.Description != "Meta description here" {
		t.Errorf("Description = %q, want %q (fallback from meta name=description)", og.Description, "Meta description here")
	}
}

func TestParseOGMeta_OGOverridesTitle(t *testing.T) {
	htmlContent := `<html><head>
<title>Fallback</title>
<meta property="og:title" content="OG Title" />
</head><body></body></html>`

	og, err := ParseOGMeta(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.Title != "OG Title" {
		t.Errorf("Title = %q, want %q (og:title should override <title>)", og.Title, "OG Title")
	}
}

func TestParseOGMeta_OGDescriptionOverridesMeta(t *testing.T) {
	htmlContent := `<html><head>
<meta name="description" content="plain desc" />
<meta property="og:description" content="OG desc" />
</head><body></body></html>`

	og, err := ParseOGMeta(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.Description != "OG desc" {
		t.Errorf("Description = %q, want %q", og.Description, "OG desc")
	}
}

func TestParseOGMeta_EmptyHTML(t *testing.T) {
	og, err := ParseOGMeta(strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.Title != "" || og.Description != "" {
		t.Errorf("expected empty results, got title=%q desc=%q", og.Title, og.Description)
	}
}

func TestParseOGMeta_StopsAtBody(t *testing.T) {
	htmlContent := `<html><head>
<meta property="og:title" content="Head Title" />
</head>
<body>
<meta property="og:description" content="Should Not Be Parsed" />
</body></html>`

	og, err := ParseOGMeta(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.Title != "Head Title" {
		t.Errorf("Title = %q, want %q", og.Title, "Head Title")
	}
	if og.Description != "" {
		t.Errorf("Description should be empty (meta after <body>), got %q", og.Description)
	}
}

func TestParseOGMeta_ChineseContent(t *testing.T) {
	htmlContent := `<html><head>
<meta property="og:title" content="新闻标题" />
<meta property="og:description" content="这是一段中文描述" />
<meta property="og:site_name" content="新闻网" />
</head><body></body></html>`

	og, err := ParseOGMeta(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.Title != "新闻标题" {
		t.Errorf("Title = %q, want %q", og.Title, "新闻标题")
	}
	if og.Description != "这是一段中文描述" {
		t.Errorf("Description = %q, want %q", og.Description, "这是一段中文描述")
	}
	if og.SiteName != "新闻网" {
		t.Errorf("SiteName = %q, want %q", og.SiteName, "新闻网")
	}
}

func TestIsPrivateHost(t *testing.T) {
	tests := []struct {
		host    string
		private bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"10.0.0.1", true},
		{"192.168.1.1", true},
		{"172.16.0.1", true},
		{"", true},
		{"8.8.8.8", false},
		{"google.com", false},
		{"example.com", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := IsPrivateHost(tt.host)
			if got != tt.private {
				t.Errorf("IsPrivateHost(%q) = %v, want %v", tt.host, got, tt.private)
			}
		})
	}
}

func TestParseOGMeta_RealWorldLike(t *testing.T) {
	htmlContent := `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<meta property="og:site_name" content="YouTube">
<meta property="og:url" content="https://www.youtube.com/watch?v=dQw4w9WgXcQ">
<meta property="og:title" content="Rick Astley - Never Gonna Give You Up (Official Music Video)">
<meta property="og:image" content="https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg">
<meta property="og:description" content="The official video for &quot;Never Gonna Give You Up&quot; by Rick Astley">
<meta property="og:type" content="video.other">
<title>Rick Astley - Never Gonna Give You Up - YouTube</title>
</head>
<body>lots of content here</body>
</html>`

	og, err := ParseOGMeta(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("ParseOGMeta error: %v", err)
	}
	if og.SiteName != "YouTube" {
		t.Errorf("SiteName = %q, want YouTube", og.SiteName)
	}
	if og.Title != "Rick Astley - Never Gonna Give You Up (Official Music Video)" {
		t.Errorf("Title = %q", og.Title)
	}
	if og.Type != "video.other" {
		t.Errorf("Type = %q, want video.other", og.Type)
	}
	if og.Image != "https://i.ytimg.com/vi/dQw4w9WgXcQ/maxresdefault.jpg" {
		t.Errorf("Image = %q", og.Image)
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"https://example.com", "https://example.com", false},
		{"http://example.com", "http://example.com", false},
		{"example.com", "https://example.com", false},
		{"www.example.com/path", "https://www.example.com/path", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, parsed, err := NormalizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
			if parsed == nil && !tt.wantErr {
				t.Errorf("NormalizeURL(%q) returned nil parsed URL", tt.input)
			}
		})
	}
}

func TestIsImageURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com/photo.jpg", true},
		{"https://example.com/photo.jpeg", true},
		{"https://example.com/photo.png", true},
		{"https://example.com/photo.gif", true},
		{"https://example.com/photo.webp", true},
		{"https://example.com/photo.JPG", true},
		{"https://example.com/photo.jpg?w=800&h=600", true},
		{"https://example.com/photo.jpg#anchor", true},
		{"https://example.com/page.html", false},
		{"https://example.com/video.mp4", false},
		{"https://example.com/path/to/page", false},
		{"https://example.com/", false},
		{"not-a-url", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := IsImageURL(tt.url)
			if got != tt.want {
				t.Errorf("IsImageURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestResolveImageURL(t *testing.T) {
	tests := []struct {
		name     string
		pageURL  string
		imageURL string
		want     string
	}{
		{"absolute URL unchanged", "https://example.com/page", "https://cdn.example.com/img.jpg", "https://cdn.example.com/img.jpg"},
		{"http absolute URL unchanged", "https://example.com/page", "http://cdn.example.com/img.jpg", "http://cdn.example.com/img.jpg"},
		{"relative path", "https://example.com/articles/page", "/images/og.jpg", "https://example.com/images/og.jpg"},
		{"relative no slash", "https://example.com/articles/page", "images/og.jpg", "https://example.com/articles/images/og.jpg"},
		{"empty image URL", "https://example.com/page", "", ""},
		{"protocol-relative", "https://example.com/page", "//cdn.example.com/img.jpg", "https://cdn.example.com/img.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveImageURL(tt.pageURL, tt.imageURL)
			if got != tt.want {
				t.Errorf("ResolveImageURL(%q, %q) = %q, want %q", tt.pageURL, tt.imageURL, got, tt.want)
			}
		})
	}
}
