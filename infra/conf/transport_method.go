package conf

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/platform/filesystem"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/common/utils"
	"github.com/xtls/xray-core/transport/internet/headers/http"
	"github.com/xtls/xray-core/transport/internet/headers/noop"
	"github.com/xtls/xray-core/transport/internet/hysteria"
	"github.com/xtls/xray-core/transport/internet/tcp"
	"github.com/xtls/xray-core/transport/internet/websocket"
	"google.golang.org/protobuf/proto"
)

type NoOpConnectionAuthenticator struct{}

func (NoOpConnectionAuthenticator) Build() (proto.Message, error) {
	return new(noop.ConnectionConfig), nil
}

type AuthenticatorRequest struct {
	Version string                 `json:"version"`
	Method  string                 `json:"method"`
	Path    StringList             `json:"path"`
	Headers map[string]*StringList `json:"headers"`
}

func sortMapKeys(m map[string]*StringList) []string {
	var keys []string
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (v *AuthenticatorRequest) Build() (*http.RequestConfig, error) {
	config := &http.RequestConfig{
		Uri: []string{"/"},
		Header: []*http.Header{
			{
				Name:  "Host",
				Value: []string{"www.baidu.com", "www.bing.com"},
			},
			{
				Name:  "User-Agent",
				Value: []string{utils.ChromeUA},
			},
			{
				Name:  "Sec-CH-UA",
				Value: []string{utils.ChromeUACH},
			},
			{
				Name:  "Sec-CH-UA-Mobile",
				Value: []string{"?0"},
			},
			{
				Name:  "Sec-CH-UA-Platform",
				Value: []string{"Windows"},
			},
			{
				Name:  "Sec-Fetch-Mode",
				Value: []string{"no-cors", "cors", "same-origin"},
			},
			{
				Name:  "Sec-Fetch-Dest",
				Value: []string{"empty"},
			},
			{
				Name:  "Sec-Fetch-Site",
				Value: []string{"none"},
			},
			{
				Name:  "Sec-Fetch-User",
				Value: []string{"?1"},
			},
			{
				Name:  "Accept-Encoding",
				Value: []string{"gzip, deflate"},
			},
			{
				Name:  "Connection",
				Value: []string{"keep-alive"},
			},
			{
				Name:  "Pragma",
				Value: []string{"no-cache"},
			},
		},
	}

	if len(v.Version) > 0 {
		config.Version = &http.Version{Value: v.Version}
	}

	if len(v.Method) > 0 {
		config.Method = &http.Method{Value: v.Method}
	}

	if len(v.Path) > 0 {
		config.Uri = append([]string(nil), v.Path...)
	}

	if len(v.Headers) > 0 {
		config.Header = make([]*http.Header, 0, len(v.Headers))
		headerNames := sortMapKeys(v.Headers)
		for _, key := range headerNames {
			value := v.Headers[key]
			if value == nil {
				return nil, errors.New("empty HTTP header value: " + key).AtError()
			}
			config.Header = append(config.Header, &http.Header{
				Name:  key,
				Value: append([]string(nil), (*value)...),
			})
		}
	}

	return config, nil
}

type AuthenticatorResponse struct {
	Version string                 `json:"version"`
	Status  string                 `json:"status"`
	Reason  string                 `json:"reason"`
	Headers map[string]*StringList `json:"headers"`
}

func (v *AuthenticatorResponse) Build() (*http.ResponseConfig, error) {
	config := &http.ResponseConfig{
		Header: []*http.Header{
			{
				Name:  "Content-Type",
				Value: []string{"application/octet-stream", "video/mpeg"},
			},
			{
				Name:  "Transfer-Encoding",
				Value: []string{"chunked"},
			},
			{
				Name:  "Connection",
				Value: []string{"keep-alive"},
			},
			{
				Name:  "Pragma",
				Value: []string{"no-cache"},
			},
			{
				Name:  "Cache-Control",
				Value: []string{"private", "no-cache"},
			},
		},
	}

	if len(v.Version) > 0 {
		config.Version = &http.Version{Value: v.Version}
	}

	if len(v.Status) > 0 || len(v.Reason) > 0 {
		config.Status = &http.Status{
			Code:   "200",
			Reason: "OK",
		}
		if len(v.Status) > 0 {
			config.Status.Code = v.Status
		}
		if len(v.Reason) > 0 {
			config.Status.Reason = v.Reason
		}
	}

	if len(v.Headers) > 0 {
		config.Header = make([]*http.Header, 0, len(v.Headers))
		headerNames := sortMapKeys(v.Headers)
		for _, key := range headerNames {
			value := v.Headers[key]
			if value == nil {
				return nil, errors.New("empty HTTP header value: " + key).AtError()
			}
			config.Header = append(config.Header, &http.Header{
				Name:  key,
				Value: append([]string(nil), (*value)...),
			})
		}
	}

	return config, nil
}

type Authenticator struct {
	Request  AuthenticatorRequest  `json:"request"`
	Response AuthenticatorResponse `json:"response"`
}

func (v *Authenticator) Build() (proto.Message, error) {
	config := new(http.Config)
	requestConfig, err := v.Request.Build()
	if err != nil {
		return nil, err
	}
	config.Request = requestConfig

	responseConfig, err := v.Response.Build()
	if err != nil {
		return nil, err
	}
	config.Response = responseConfig

	return config, nil
}

var tcpHeaderLoader = NewJSONConfigLoader(ConfigCreatorCache{
	"none": func() interface{} { return new(NoOpConnectionAuthenticator) },
	"http": func() interface{} { return new(Authenticator) },
}, "type", "")

type TCPConfig struct {
	HeaderConfig        json.RawMessage `json:"header"`
	AcceptProxyProtocol bool            `json:"acceptProxyProtocol"`
}

// Build implements Buildable.
func (c *TCPConfig) Build() (proto.Message, error) {
	config := new(tcp.Config)
	if len(c.HeaderConfig) > 0 {
		headerConfig, _, err := tcpHeaderLoader.Load(c.HeaderConfig)
		if err != nil {
			return nil, errors.New("invalid TCP header config").Base(err).AtError()
		}
		ts, err := headerConfig.(Buildable).Build()
		if err != nil {
			return nil, errors.New("invalid TCP header config").Base(err).AtError()
		}
		config.HeaderSettings = serial.ToTypedMessage(ts)
	}
	if c.AcceptProxyProtocol {
		config.AcceptProxyProtocol = c.AcceptProxyProtocol
	}
	return config, nil
}

type WebSocketConfig struct {
	Host                string            `json:"host"`
	Path                string            `json:"path"`
	Headers             map[string]string `json:"headers"`
	AcceptProxyProtocol bool              `json:"acceptProxyProtocol"`
	HeartbeatPeriod     uint32            `json:"heartbeatPeriod"`
}

// Build implements Buildable.
func (c *WebSocketConfig) Build() (proto.Message, error) {
	path := c.Path
	var ed uint32
	if u, err := url.Parse(path); err == nil {
		if q := u.Query(); q.Get("ed") != "" {
			Ed, _ := strconv.Atoi(q.Get("ed"))
			ed = uint32(Ed)
			q.Del("ed")
			u.RawQuery = q.Encode()
			path = u.String()
		}
	}
	// Priority (client): host > serverName > address
	for k, v := range c.Headers {
		if strings.ToLower(k) == "host" {
			errors.PrintDeprecatedFeatureWarning(`"host" in "headers"`, `independent "host"`)
			if c.Host == "" {
				c.Host = v
			}
			delete(c.Headers, k)
		}
	}
	config := &websocket.Config{
		Path:                path,
		Host:                c.Host,
		Header:              c.Headers,
		AcceptProxyProtocol: c.AcceptProxyProtocol,
		Ed:                  ed,
		HeartbeatPeriod:     c.HeartbeatPeriod,
	}
	return config, nil
}

const (
	Byte     = 1
	Kilobyte = 1024 * Byte
	Megabyte = 1024 * Kilobyte
	Gigabyte = 1024 * Megabyte
	Terabyte = 1024 * Gigabyte
)

type Bandwidth string

func (b Bandwidth) Bps() (uint64, error) {
	s := strings.TrimSpace(strings.ToLower(string(b)))
	if s == "" {
		return 0, nil
	}

	idx := len(s)
	for i, c := range s {
		if (c < '0' || c > '9') && c != '.' {
			idx = i
			break
		}
	}

	numStr := s[:idx]
	unit := strings.TrimSpace(s[idx:])

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}

	mul := uint64(1)
	switch unit {
	case "", "b", "bps":
		mul = Byte
	case "k", "kb", "kbps":
		mul = Kilobyte
	case "m", "mb", "mbps":
		mul = Megabyte
	case "g", "gb", "gbps":
		mul = Gigabyte
	case "t", "tb", "tbps":
		mul = Terabyte
	default:
		return 0, errors.New("unsupported unit: " + unit)
	}

	return uint64(val*float64(mul)) / 8, nil
}

type UdpHop struct {
	PortList PortList   `json:"ports"`
	Interval Int32Range `json:"interval"`
}

type Masquerade struct {
	Type string `json:"type"`

	Dir string `json:"dir"`

	Url         string `json:"url"`
	RewriteHost bool   `json:"rewriteHost"`
	Insecure    bool   `json:"insecure"`

	Content    string            `json:"content"`
	Headers    map[string]string `json:"headers"`
	StatusCode int32             `json:"statusCode"`
}

type HysteriaConfig struct {
	Version int32  `json:"version"`
	Auth    string `json:"auth"`

	Congestion *string    `json:"congestion"`
	Up         *Bandwidth `json:"up"`
	Down       *Bandwidth `json:"down"`
	UdpHop     *UdpHop    `json:"udphop"`

	UdpIdleTimeout int64      `json:"udpIdleTimeout"`
	Masquerade     Masquerade `json:"masquerade"`
}

func (c *HysteriaConfig) Build() (proto.Message, error) {
	if c.Version != 2 {
		return nil, errors.New("version != 2")
	}

	if c.Congestion != nil || c.Up != nil || c.Down != nil || c.UdpHop != nil {
		errors.LogWarning(context.Background(), "congestion & up & down & udphop move to finalmask/quicParams")
	}

	if c.UdpIdleTimeout != 0 && (c.UdpIdleTimeout < 2 || c.UdpIdleTimeout > 600) {
		return nil, errors.New("UdpIdleTimeout must be between 2 and 600")
	}

	config := &hysteria.Config{}
	config.Auth = c.Auth
	config.UdpIdleTimeout = c.UdpIdleTimeout
	config.MasqType = c.Masquerade.Type
	config.MasqFile = c.Masquerade.Dir
	config.MasqUrl = c.Masquerade.Url
	config.MasqUrlRewriteHost = c.Masquerade.RewriteHost
	config.MasqUrlInsecure = c.Masquerade.Insecure
	config.MasqString = c.Masquerade.Content
	config.MasqStringHeaders = c.Masquerade.Headers
	config.MasqStringStatusCode = c.Masquerade.StatusCode

	if config.UdpIdleTimeout == 0 {
		config.UdpIdleTimeout = 60
	}

	return config, nil
}

func readFileOrString(f string, s []string) ([]byte, error) {
	if len(f) > 0 {
		return filesystem.ReadCert(f)
	}
	if len(s) > 0 {
		return []byte(strings.Join(s, "\n")), nil
	}
	return nil, errors.New("both file and bytes are empty.")
}
