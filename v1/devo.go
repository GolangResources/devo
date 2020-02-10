package devo

import (
	"strings"
	"strconv"
	"errors"
	"log"
	"fmt"
	"net/http"
	"encoding/json"
	"time"
	"crypto/sha256"
	"crypto/hmac"
	"crypto/tls"
	"bufio"
	"regexp"
)

type DevoClient struct {
	APIKey		string
	APISecret	string
	SerreaURL	string
	Debug		bool
	HTTPClient	*http.Client
	BufferSize	int
}

type DevoRequest struct {
	DateFrom	int64		`json:"from"`
	DateTo		int64		`json:"to"`
	Query		string		`json:"query"`
	Mode		SMode		`json:"mode"`
}

type SMode struct {
	Type		string		`json:"type"`
}

type SDevoResponse struct {
	Message		string		`json:"msg"`
	Timestamp	int64		`json:"timestamp"`
	CID		string		`json:"cid"`
	Status		int		`json:"status"`
	Object		[]interface{}	`json:"object"`
}

func Init(sp *DevoClient) DevoClient {
	var s DevoClient
	if sp != nil {
		s = *sp
	} else {
		s = DevoClient{
			Debug: false,
		}
	}
	// Customize the Transport to have larger connection pool
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport := *defaultTransportPointer // dereference it to get a copy of the struct that the pointer points to
	defaultTransport.MaxIdleConns = 100
	defaultTransport.MaxIdleConnsPerHost = 100
	defaultTransport.Proxy = nil
	//defaultTransport.Proxy = http.ProxyURL(proxyURL)
	defaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	wc := &http.Client{
		Transport: &defaultTransport,
		Timeout: 300 * time.Second,
	}
	s.HTTPClient = wc
	return s
}

func (s *DevoClient) QueryRaw(from int64, to int64, query string, resultmsg chan string) (lastLine string, err error) {
	var q DevoRequest
	q.Query = query
	q.DateFrom = from
	q.DateTo = to
	// Set mode
	q.Mode.Type = "json"
	json_msg, err := json.Marshal(q)
	if s.Debug {
		fmt.Println("json_msg:", string(json_msg))
	}
	if err != nil {
		log.Println("query marshal err:", err)
		return "", err
	}
	// Set timestamp
	tstamp := int64(time.Now().UnixNano()/1000000)
	tstamp_ := strconv.FormatInt(tstamp, 10)
	message := s.APIKey + string(json_msg) + tstamp_
	mac := hmac.New(sha256.New, []byte(s.APISecret))
	mac.Write([]byte(message))
	expectedMAC := mac.Sum(nil)
	sign_hex := fmt.Sprintf("%x", expectedMAC)
	if s.Debug {
		fmt.Println("tstamp_:", tstamp_)
		fmt.Println("sign_hex:", string(sign_hex))
	}
	// Prepare http request
	data := strings.NewReader(string(json_msg))
	req, err := http.NewRequest("POST", s.SerreaURL, data)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-logtrust-apikey", s.APIKey)
	req.Header.Set("x-logtrust-timestamp", tstamp_)
	req.Header.Set("x-logtrust-sign", string(sign_hex))
	// Make http request
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		log.Println("Error in client.Do:", err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Println("Query response err:", resp.Status)
		return "", errors.New("query response status: " + resp.Status)
	}
	if s.Debug {
		fmt.Println("resp header:", resp.Header)
		fmt.Println("resp cookies:", resp.Header["Cookies"])
	}
	ch := false
	for _, v := range resp.TransferEncoding {
		if v == "chunked" {
			ch = true
		}
	}
	if ch == true {
		if s.Debug {
			log.Println("resp", resp)
		}
		respio := resp.Body
		scanner := bufio.NewScanner(respio)
		if (s.BufferSize != 4096 && s.BufferSize != 0) {
			buf := make([]byte, s.BufferSize)
			scanner.Buffer(buf, s.BufferSize)
		}
		split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			if atEOF && len(data) == 0 {
				return 0, nil, nil
			}
			if i := strings.Index(string(data), "},{"); i >= 0 {
				//return i + 1, data[0:i], nil
				return i + 1, data[1:i+1], nil
			}
			if atEOF {
				return len(data), data, nil
			}

			return
		}
		scanner.Split(split)
		lh := false
		re := regexp.MustCompile(`(^.*,"object":\[)`)
		for scanner.Scan() {
			lastLine = scanner.Text()
			if (lh == false) {
				resultmsg <- re.ReplaceAllString(lastLine, "")
				lh = true
			} else {
				resultmsg <- lastLine
			}
		}
	}
	return lastLine, nil
}
func (s *DevoClient) ContinuousQuery (from int64, query string, resultmsg chan string) (err error) {
	var fromD int64
	fromD = from
	re := regexp.MustCompile(`(^,{)`)
	for {
		if s.Debug {
			log.Println("DEBUG: QueryRAW", fromD, query)
		}
		lastLine, err := s.QueryRaw(fromD, time.Unix(from, 0).AddDate(1, 0, 0).Unix(), query, resultmsg)
		if (err != nil) {
			return err
		}
		lastLineMsg := make(map[string]interface{})
		lastLine = re.ReplaceAllString(lastLine, "{")
		err = json.Unmarshal([]byte(lastLine), &lastLineMsg)
		if (err != nil) {
			return err
		}
		if s.Debug {
			log.Println("DEBUG: Lastline",lastLine,"fromD",fromD, "eventDate", int64(lastLineMsg["eventdate"].(float64)))
		}
		fs := fmt.Sprintf("%f", lastLineMsg["eventdate"].(float64))
		fromD, _ = strconv.ParseInt(fs[0:10], 10, 64)
	}
}
