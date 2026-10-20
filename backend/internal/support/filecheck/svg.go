package filecheck

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strings"
)

var blockedElements = map[string]bool{
	"script": true, "foreignobject": true, "iframe": true, "object": true, "embed": true,
	"audio": true, "video": true, "handler": true, "listener": true, "set": true,
	"animate": true, "animatemotion": true, "animatetransform": true, "discard": true,
}

var (
	safeDataImage = regexp.MustCompile(`^data:image/(png|jpeg|gif|webp);base64,[A-Za-z0-9+/=\s]+$`)
	cssURL        = regexp.MustCompile(`(?i)url\(\s*['"]?\s*([^'")\s]*)`)
)

// SanitizeSVG removes scripts, event handlers, external references and foreignObject from an SVG document.
func SanitizeSVG(data []byte) ([]byte, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	var out bytes.Buffer
	depth, skipDepth := 0, 0
	sawRoot := false
	var styleBuf *bytes.Buffer
	var styleStart xml.StartElement
	for {
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, Rejected("The SVG file could not be parsed.")
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			name := strings.ToLower(t.Name.Local)
			if !sawRoot {
				if name != "svg" {
					return nil, InvalidType("The file is not an SVG image.")
				}
				sawRoot = true
			}
			if skipDepth > 0 {
				continue
			}
			if blockedElements[name] {
				skipDepth = depth
				continue
			}
			if name == "style" {
				styleBuf, styleStart = &bytes.Buffer{}, t
				continue
			}
			writeStart(&out, t)
		case xml.EndElement:
			if skipDepth > 0 {
				if depth == skipDepth {
					skipDepth = 0
				}
				depth--
				continue
			}
			depth--
			if styleBuf != nil && strings.EqualFold(t.Name.Local, "style") {
				if safeCSS(styleBuf.String()) {
					writeStart(&out, styleStart)
					escapeText(&out, styleBuf.Bytes())
					writeEnd(&out, t)
				}
				styleBuf = nil
				continue
			}
			writeEnd(&out, t)
		case xml.CharData:
			if skipDepth > 0 || !sawRoot {
				continue
			}
			if styleBuf != nil {
				styleBuf.Write(t)
				continue
			}
			escapeText(&out, t)
		}
	}
	if !sawRoot || depth != 0 {
		return nil, Rejected("The SVG file could not be parsed.")
	}
	return out.Bytes(), nil
}

func writeStart(out *bytes.Buffer, t xml.StartElement) {
	out.WriteByte('<')
	out.WriteString(qualified(t.Name))
	for _, a := range t.Attr {
		if !safeAttr(a) {
			continue
		}
		out.WriteByte(' ')
		out.WriteString(qualified(a.Name))
		out.WriteString(`="`)
		escapeText(out, []byte(a.Value))
		out.WriteByte('"')
	}
	out.WriteByte('>')
}

func writeEnd(out *bytes.Buffer, t xml.EndElement) {
	out.WriteString("</")
	out.WriteString(qualified(t.Name))
	out.WriteByte('>')
}

func escapeText(out *bytes.Buffer, b []byte) {
	_ = xml.EscapeText(out, b)
}

func qualified(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

func safeAttr(a xml.Attr) bool {
	local := strings.ToLower(a.Name.Local)
	value := strings.TrimSpace(a.Value)
	lowerValue := strings.ToLower(value)
	switch {
	case strings.HasPrefix(local, "on"):
		return false
	case local == "href" || local == "src" || local == "action" || local == "formaction":
		return strings.HasPrefix(value, "#") || safeDataImage.MatchString(value)
	case strings.Contains(lowerValue, "javascript:") || strings.Contains(lowerValue, "vbscript:"):
		return false
	case local == "style":
		return safeCSS(value)
	}
	return !strings.Contains(lowerValue, "url(") || safeCSS(value)
}

func safeCSS(css string) bool {
	lower := strings.ToLower(css)
	if strings.Contains(lower, "@import") || strings.Contains(lower, "expression(") || strings.Contains(lower, "javascript:") {
		return false
	}
	for _, m := range cssURL.FindAllStringSubmatch(css, -1) {
		if !strings.HasPrefix(m[1], "#") {
			return false
		}
	}
	return true
}
