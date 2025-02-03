package aad

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/upmux/tenantfinder/pkg/session"
	"github.com/upmux/tenantfinder/pkg/source"
)

// SOAP request structure
type Envelope struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	SoapNS  string   `xml:"xmlns:soap,attr"`
	ExmNS   string   `xml:"xmlns:exm,attr"`
	ExtNS   string   `xml:"xmlns:ext,attr"`
	ANS     string   `xml:"xmlns:a,attr"`
	XsiNS   string   `xml:"xmlns:xsi,attr"`
	XsdNS   string   `xml:"xmlns:xsd,attr"`
	Header  Header   `xml:"soap:Header"`
	Body    Body     `xml:"soap:Body"`
}

type Header struct {
	Action  Action  `xml:"a:Action"`
	To      To      `xml:"a:To"`
	ReplyTo ReplyTo `xml:"a:ReplyTo"`
}

type Action struct {
	MustUnderstand string `xml:"soap:mustUnderstand,attr"`
	Value          string `xml:",chardata"`
}

type To struct {
	MustUnderstand string `xml:"soap:mustUnderstand,attr"`
	Value          string `xml:",chardata"`
}

type ReplyTo struct {
	Address string `xml:"a:Address"`
}

type Body struct {
	GetFederationInfo GetFederationInfo `xml:"GetFederationInformationRequestMessage"`
}

type GetFederationInfo struct {
	XMLNs   string  `xml:"xmlns,attr"`
	Request Request `xml:"Request"`
}

type Request struct {
	Domain string `xml:"Domain"`
}

// SOAP response structure
type ResponseEnvelope struct {
	XMLName xml.Name       `xml:"Envelope"`
	Header  ResponseHeader `xml:"Header"`
	Body    ResponseBody   `xml:"Body"`
}

type ResponseHeader struct {
	Action            string            `xml:"Action"`
	ServerVersionInfo ServerVersionInfo `xml:"ServerVersionInfo"`
}

type ServerVersionInfo struct {
	MajorVersion     string `xml:"MajorVersion"`
	MinorVersion     string `xml:"MinorVersion"`
	MajorBuildNumber string `xml:"MajorBuildNumber"`
	MinorBuildNumber string `xml:"MinorBuildNumber"`
	Version          string `xml:"Version"`
}

type ResponseBody struct {
	GetFederationInfoResponse GetFederationInfoResponse `xml:"GetFederationInformationResponseMessage"`
}

type GetFederationInfoResponse struct {
	Response Response `xml:"Response"`
}

type Response struct {
	ErrorCode      string       `xml:"ErrorCode"`
	ErrorMessage   string       `xml:"ErrorMessage"`
	ApplicationUri string       `xml:"ApplicationUri"`
	Domains        Domains      `xml:"Domains"`
	TokenIssuers   TokenIssuers `xml:"TokenIssuers"`
}

type Domains struct {
	Domain []string `xml:"Domain"`
}

type TokenIssuers struct {
	TokenIssuer []TokenIssuer `xml:"TokenIssuer"`
}

type TokenIssuer struct {
	Endpoint string `xml:"Endpoint"`
	Uri      string `xml:"Uri"`
}

type Source struct {
	timeTaken time.Duration
	errors    int
	results   int
}

func (s *Source) Run(ctx context.Context, domain string, sess *session.Session) <-chan source.Result {
	results := make(chan source.Result)
	s.errors = 0
	s.results = 0

	go func() {
		defer func(startTime time.Time) {
			s.timeTaken = time.Since(startTime)
			close(results)
		}(time.Now())

		domains, err := s.fetchDomains(ctx, sess, domain)
		if err != nil {
			results <- source.Result{
				Source: s.Name(),
				Type:   source.Error,
				Error:  fmt.Errorf("failed to fetch domains: %v", err),
			}
			s.errors++
			return
		}

		for _, domain := range domains {
			results <- source.Result{
				Source: s.Name(),
				Type:   source.Domain,
				Value:  domain,
			}
			s.results++
		}
	}()

	return results
}

func (s *Source) fetchDomains(ctx context.Context, sess *session.Session, rootUrl string) ([]string, error) {
	envelope := &Envelope{
		SoapNS: "http://schemas.xmlsoap.org/soap/envelope/",
		ExmNS:  "http://schemas.microsoft.com/exchange/services/2006/messages",
		ExtNS:  "http://schemas.microsoft.com/exchange/services/2006/types",
		ANS:    "http://www.w3.org/2005/08/addressing",
		XsiNS:  "http://www.w3.org/2001/XMLSchema-instance",
		XsdNS:  "http://www.w3.org/2001/XMLSchema",
		Header: Header{
			Action: Action{
				MustUnderstand: "1",
				Value:          "http://schemas.microsoft.com/exchange/2010/Autodiscover/Autodiscover/GetFederationInformation",
			},
			To: To{
				MustUnderstand: "1",
				Value:          "https://autodiscover-s.outlook.com/autodiscover/autodiscover.svc",
			},
			ReplyTo: ReplyTo{
				Address: "http://www.w3.org/2005/08/addressing/anonymous",
			},
		},
		Body: Body{
			GetFederationInfo: GetFederationInfo{
				XMLNs: "http://schemas.microsoft.com/exchange/2010/Autodiscover",
				Request: Request{
					Domain: rootUrl, // Use the target domain from the Run method
				},
			},
		},
	}

	xmlData, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SOAP request: %v", err)
	}

	// Add XML declaration
	xmlString := `<?xml version="1.0" encoding="utf-8"?>` + "\n" + string(xmlData)

	// Update endpoint URL
	url := "https://autodiscover-s.outlook.com/autodiscover/autodiscover.svc"

	headers := map[string]string{
		"User-Agent":   "AutodiscoverClient",
		"Content-Type": "text/xml; charset=utf-8",
		"SOAPAction":   "http://schemas.microsoft.com/exchange/2010/Autodiscover/Autodiscover/GetFederationInformation",
	}

	resp, err := sess.Post(ctx, url, "", headers, strings.NewReader(xmlString))
	if err != nil {
		return nil, fmt.Errorf("failed to send SOAP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var response ResponseEnvelope
	if err := xml.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if response.Body.GetFederationInfoResponse.Response.ErrorCode != "NoError" {
		return nil, fmt.Errorf("federation info request failed: %s", response.Body.GetFederationInfoResponse.Response.ErrorMessage)
	}

	return response.Body.GetFederationInfoResponse.Response.Domains.Domain, nil
}

func (s *Source) Name() string {
	return "aad"
}

func (s *Source) IsDefault() bool {
	return true
}

func (s *Source) NeedsKey() bool {
	return false
}

func (s *Source) AddApiKeys(_ []string) {
	// No API keys needed
}

func (s *Source) Statistics() source.Statistics {
	return source.Statistics{
		Errors:    s.errors,
		Results:   s.results,
		TimeTaken: s.timeTaken,
	}
}
