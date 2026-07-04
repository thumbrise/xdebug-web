package xdebug

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/thumbrise/xdebug-web/pkg/plugins"
)

var isoRegexp = regexp.MustCompile(`encoding="iso-8859-1"`)

func fixEncoding(data []byte) []byte {
	return isoRegexp.ReplaceAll(data, []byte(`encoding="UTF-8"`))
}

type initPacket struct {
	XMLName  xml.Name `xml:"init"`
	Language string   `xml:"language,attr"`
	FileURI  string   `xml:"fileuri,attr"`
	AppID    string   `xml:"appid,attr"`
	IdeKey   string   `xml:"idekey,attr"`
}

type initInfo struct {
	Language string
	FileURI  string
	AppID    string
	IdeKey   string
}

func parseInit(data []byte) (*initInfo, error) {
	data = fixEncoding(data)

	var p initPacket
	if err := xml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse init: %w", err)
	}

	return &initInfo{
		Language: p.Language,
		FileURI:  p.FileURI,
		AppID:    p.AppID,
		IdeKey:   p.IdeKey,
	}, nil
}

type stepResponse struct {
	XMLName xml.Name `xml:"response"`
	Status  string   `xml:"status,attr"`
	Reason  string   `xml:"reason,attr"`

	Message *struct {
		Filename string `xml:"filename,attr"`
		Lineno   int    `xml:"lineno,attr"`
	} `xml:"https://xdebug.org/dbgp/xdebug message"`
}

type stepResult struct {
	Status   plugins.Status
	Filename string
	Lineno   int
	Reason   string
}

func parseStepResponse(data []byte) (*stepResult, error) {
	data = fixEncoding(data)

	var resp stepResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse step response: %w", err)
	}

	result := &stepResult{
		Status: plugins.Status(resp.Status),
		Reason: resp.Reason,
	}

	if resp.Message != nil {
		result.Filename = resp.Message.Filename
		result.Lineno = resp.Message.Lineno
	}

	return result, nil
}

type breakpointSetResponse struct {
	XMLName xml.Name `xml:"response"`
	ID      string   `xml:"id,attr"`
}

func parseBreakpointSetResponse(data []byte) (string, error) {
	data = fixEncoding(data)

	var resp breakpointSetResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse breakpoint_set: %w", err)
	}

	if resp.ID == "" {
		return "", plugins.ErrMissingBreakpointID
	}

	return resp.ID, nil
}

type contextGetResponse struct {
	XMLName    xml.Name          `xml:"response"`
	Properties []contextProperty `xml:"property"`
}

type contextProperty struct {
	Name        string `xml:"name,attr"`
	Type        string `xml:"type,attr"`
	Encoding    string `xml:"encoding,attr"`
	ClassName   string `xml:"classname,attr"`
	NumChildren int    `xml:"numchildren,attr"`
	Value       string `xml:",chardata"`
}

func parseContextGetResponse(data []byte) ([]plugins.Variable, error) {
	data = fixEncoding(data)

	var resp contextGetResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse context_get: %w", err)
	}

	vars := make([]plugins.Variable, 0, len(resp.Properties))

	for _, p := range resp.Properties {
		v := plugins.Variable{Name: p.Name, Type: p.Type}
		v.Value = decodeValue(p.Value, p.Encoding, p.Type)

		if p.NumChildren > 0 {
			v.ClassName = p.ClassName
			v.NumChildren = p.NumChildren
		}

		vars = append(vars, v)
	}

	return vars, nil
}

type propertyGetResponse struct {
	XMLName    xml.Name          `xml:"response"`
	Properties []contextProperty `xml:"property"`
}

func parsePropertyGetResponse(data []byte) ([]plugins.Variable, error) {
	data = fixEncoding(data)

	var resp propertyGetResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse property_get: %w", err)
	}

	vars := make([]plugins.Variable, 0, len(resp.Properties))

	for _, p := range resp.Properties {
		v := plugins.Variable{Name: p.Name, Type: p.Type}
		v.Value = decodeValue(p.Value, p.Encoding, p.Type)

		if p.NumChildren > 0 {
			v.ClassName = p.ClassName
			v.NumChildren = p.NumChildren
		}

		vars = append(vars, v)
	}

	return vars, nil
}

type stackGetResponse struct {
	XMLName xml.Name     `xml:"response"`
	Frames  []stackFrame `xml:"stack"`
}

type stackFrame struct {
	Level    int    `xml:"level,attr"`
	Filename string `xml:"filename,attr"`
	Lineno   int    `xml:"lineno,attr"`
	Where    string `xml:"where,attr"`
}

func parseStackGetResponse(data []byte) ([]plugins.Frame, error) {
	data = fixEncoding(data)

	var resp stackGetResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse stack_get: %w", err)
	}

	frames := make([]plugins.Frame, len(resp.Frames))
	for i, f := range resp.Frames {
		frames[i] = plugins.Frame(f)
	}

	return frames, nil
}

func formatCommand(cmd string, txID int, args map[string]string) string {
	var b strings.Builder

	b.WriteString(cmd)
	b.WriteString(" -i ")
	fmt.Fprintf(&b, "%d", txID)

	for k, v := range args {
		b.WriteString(" -")
		b.WriteString(k)
		b.WriteString(" ")
		b.WriteString(url.QueryEscape(v))
	}

	b.WriteByte(0)

	return b.String()
}

func formatBreakpointSetCmd(txID int, fileURI string, line int) string {
	return fmt.Sprintf("breakpoint_set -i %d -t line -f %s -n %d\x00", txID, fileURI, line)
}

func formatBreakpointRemoveCmd(txID int, bpID string) string {
	return fmt.Sprintf("breakpoint_remove -i %d -d %s\x00", txID, bpID)
}

func formatContextGetCmd(txID int, depth int) string {
	return fmt.Sprintf("context_get -i %d -d %d\x00", txID, depth)
}

func formatStackGetCmd(txID int) string {
	return fmt.Sprintf("stack_get -i %d\x00", txID)
}

func formatPropertyGetCmd(txID int, depth int, name string) string {
	return fmt.Sprintf("property_get -i %d -d %d -n %s\x00", txID, depth, url.QueryEscape(name))
}

func decodeValue(raw, encoding, typ string) string {
	raw = strings.TrimSpace(raw)

	if encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return raw
		}

		raw = string(decoded)
	}

	switch typ {
	case "null":
		return "null"
	case "boolean":
		if raw == "1" || raw == "true" {
			return "true"
		}

		return "false"
	default:
		return raw
	}
}
