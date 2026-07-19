// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package htmlx

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/indaco/teseo/opengraph"
	"golang.org/x/net/html"
)

type OpenGraph struct {
	*opengraph.OpenGraphObject

	AdditionalProperties map[string]string
}

// DecodeOpengraph will parse the given byte array and return any Open Graph metadata found within. Use with existing
// HTML page data.
func DecodeOpengraph(data []byte) (*OpenGraph, error) {
	htmlData, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse data: %w", err)
	}

	og := &OpenGraph{
		AdditionalProperties: make(map[string]string),
	}

	visitNode(htmlData, og)

	return og, nil
}

// Get retrieves additional properties for OpenGraph. Returns the specified element and whether it was found.
func (og OpenGraph) Get(fieldName string) (value string, found bool) {
	if og.AdditionalProperties != nil {
		value, found = og.AdditionalProperties[fieldName]
	}
	return
}

// Set stores additional properties for OpenGraph.
func (og *OpenGraph) Set(fieldName string, value string) {
	if og.AdditionalProperties == nil {
		og.AdditionalProperties = make(map[string]string)
	}
	og.AdditionalProperties[fieldName] = value
}

// UnmarshalJSON overrides default JSON handling for OpenGraph to handle AdditionalProperties.
func (og *OpenGraph) UnmarshalJSON(b []byte) error {
	object := make(map[string]json.RawMessage)
	err := json.Unmarshal(b, &object)
	if err != nil {
		return fmt.Errorf("unmarshal opengraph: %w", err)
	}

	if raw, found := object["description"]; found {
		err = json.Unmarshal(raw, &og.Description)
		if err != nil {
			return fmt.Errorf("error reading 'description': %w", err)
		}
		delete(object, "description")
	}

	if raw, found := object["image"]; found {
		err = json.Unmarshal(raw, &og.Image)
		if err != nil {
			return fmt.Errorf("error reading 'image': %w", err)
		}
		delete(object, "image")
	}

	if raw, found := object["type"]; found {
		err = json.Unmarshal(raw, &og.Type)
		if err != nil {
			return fmt.Errorf("error reading 'object_type': %w", err)
		}
		delete(object, "object_type")
	}

	if raw, found := object["title"]; found {
		err = json.Unmarshal(raw, &og.Title)
		if err != nil {
			return fmt.Errorf("error reading 'title': %w", err)
		}
		delete(object, "title")
	}

	if raw, found := object["url"]; found {
		err = json.Unmarshal(raw, &og.URL)
		if err != nil {
			return fmt.Errorf("error reading 'url': %w", err)
		}
		delete(object, "url")
	}

	if len(object) != 0 {
		og.AdditionalProperties = make(map[string]string)
		for fieldName, fieldBuf := range object {
			var fieldVal string
			if err := json.Unmarshal(fieldBuf, &fieldVal); err != nil {
				return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
			}
			og.AdditionalProperties[fieldName] = fieldVal
		}
	}
	return nil
}

// MarshalJSON overrides default JSON handling for OpenGraph to handle AdditionalProperties.
func (og OpenGraph) MarshalJSON() ([]byte, error) {
	var err error
	object := make(map[string]json.RawMessage)

	if og.Description != "" {
		object["description"], err = json.Marshal(og.Description)
		if err != nil {
			return nil, fmt.Errorf("error marshaling 'description': %w", err)
		}
	}

	object["image"], err = json.Marshal(og.Image)
	if err != nil {
		return nil, fmt.Errorf("error marshaling 'image': %w", err)
	}

	object["type"], err = json.Marshal(og.Type)
	if err != nil {
		return nil, fmt.Errorf("error marshaling 'object_type': %w", err)
	}

	object["title"], err = json.Marshal(og.Title)
	if err != nil {
		return nil, fmt.Errorf("error marshaling 'title': %w", err)
	}

	object["url"], err = json.Marshal(og.URL)
	if err != nil {
		return nil, fmt.Errorf("error marshaling 'url': %w", err)
	}

	for fieldName, field := range og.AdditionalProperties {
		object[fieldName], err = json.Marshal(field)
		if err != nil {
			return nil, fmt.Errorf("error marshaling '%s': %w", fieldName, err)
		}
	}

	data, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("marshal opengraph: %w", err)
	}
	return data, nil
}

// UnmarshalXML implements xml.Unmarshaler. It scans all <meta> elements anywhere in the document and populates
// Opengraph fields from those whose property/name attribute starts with "og:".
func (og *OpenGraph) UnmarshalXML(dec *xml.Decoder, se xml.StartElement) error {
	if og.AdditionalProperties == nil {
		og.AdditionalProperties = make(map[string]string)
	}

	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.EndElement:
			// </head> — we're done; don't consume any further tokens.
			if strings.EqualFold(t.Name.Local, "head") {
				return nil
			}

		case xml.StartElement:
			if !strings.EqualFold(t.Name.Local, "meta") {
				// Skip non-meta elements and all their children.
				if err := dec.Skip(); err != nil {
					return err
				}
				continue
			}

			var m metaTag
			if err := dec.DecodeElement(&m, &t); err != nil {
				continue
			}

			// property takes precedence over name.
			key := m.Property
			if key == "" {
				key = m.Name
			}
			key = strings.ToLower(strings.TrimSpace(key))

			if strings.HasPrefix(key, "og:") {
				og.set(key, m.Content)
			}
		}
	}
}

func (og OpenGraph) String() string {
	var str strings.Builder
	fmt.Fprintf(&str, "<meta property=%q content=%q/>\n", "og:title", og.Title)
	fmt.Fprintf(&str, "<meta property=%q content=%q/>\n", "og:url", og.URL)
	fmt.Fprintf(&str, "<meta property=%q content=%q/>\n", "og:image", og.Image)
	fmt.Fprintf(&str, "<meta property=%q content=%q/>\n", "og:type", og.Type)
	if og.Description != "" {
		fmt.Fprintf(&str, "<meta property=%q content=%q/>\n", "og:description", og.Description)
	}
	for key, value := range og.AdditionalProperties {
		fmt.Fprintf(&str, "<meta property=%q content=%q/>\n", key, value)
	}

	return str.String()
}

// set assigns a parsed og: property to the appropriate named struct field, or additionalProperties.
func (og *OpenGraph) set(property, content string) {
	switch property {
	case "og:title":
		og.Title = content
	case "og:description":
		og.Description = content
	case "og:image":
		og.Image = content
	case "og:url":
		og.URL = content
	case "og:type":
		og.Type = content
	default:
		if og.AdditionalProperties == nil {
			og.AdditionalProperties = make(map[string]string)
		}
		og.Set(property, content)
	}
}

// metaTag is a minimal representation of a <meta> element for XML decoding.
type metaTag struct {
	Property string `xml:"property,attr"`
	Name     string `xml:"name,attr"`
	Content  string `xml:"content,attr"`
}

// visitNode recursively walks the node tree, extracting og: meta tags.
// Returns true to signal the caller to stop descending (entered <body>).
func visitNode(n *html.Node, og *OpenGraph) bool {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "body":
			return true
		case "meta":
			extractMeta(n, og)
			return false
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if visitNode(c, og) {
			return true
		}
	}
	return false
}

// extractMeta reads property/name and content attributes from a <meta> node.
func extractMeta(n *html.Node, og *OpenGraph) {
	var property, content string
	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "property", "name":
			property = strings.ToLower(a.Val)
		case "content":
			content = a.Val
		}
	}
	if strings.HasPrefix(property, "og:") {
		og.set(property, content)
	}
}
