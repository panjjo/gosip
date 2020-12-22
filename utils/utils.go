package utils

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html/charset"
)

// Error Error
type Error struct {
	err    error
	params []interface{}
}

func (err *Error) Error() string {
	if err == nil {
		return "<nil>"
	}
	str := fmt.Sprint(err.params...)
	if err.err != nil {
		str += fmt.Sprintf(" err:%s", err.err.Error())
	}
	return str
}

// NewError NewError
func NewError(err error, params ...interface{}) error {
	return &Error{err, params}
}

// JSONEncode JSONEncode
func JSONEncode(data interface{}) []byte {
	d, err := json.Marshal(data)
	if err != nil {
		logrus.Errorln("JSONEncode error:", err)
	}
	return d
}

// JSONDecode JSONDecode
func JSONDecode(data []byte, obj interface{}) error {
	return json.Unmarshal(data, obj)
}

const (
	letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandString https://github.com/kpbird/golang_random_string
func RandString(n int) string {
	output := make([]byte, n)
	// We will take n bytes, one byte for each character of output.
	randomness := make([]byte, n)
	// read all random
	_, err := rand.Read(randomness)
	if err != nil {
		panic(err)
	}
	l := len(letterBytes)
	// fill output
	for pos := range output {
		// get random item
		random := randomness[pos]
		// random % 64
		randomPos := random % uint8(l)
		// put into output
		output[pos] = letterBytes[randomPos]
	}

	return string(output)
}

func timeoutClient() *http.Client {
	connectTimeout := time.Duration(20 * time.Second)
	readWriteTimeout := time.Duration(30 * time.Second)
	return &http.Client{
		Transport: &http.Transport{
			DialContext:         timeoutDialer(connectTimeout, readWriteTimeout),
			MaxIdleConnsPerHost: 200,
			DisableKeepAlives:   true,
		},
	}
}
func timeoutDialer(cTimeout time.Duration,
	rwTimeout time.Duration) func(ctx context.Context, net, addr string) (c net.Conn, err error) {
	return func(ctx context.Context, netw, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(netw, addr, cTimeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(rwTimeout))
		return conn, nil
	}
}

// PostRequest PostRequest
func PostRequest(url string, bodyType string, body io.Reader) ([]byte, error) {
	client := timeoutClient()
	resp, err := client.Post(url, bodyType, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respbody, nil
}

// PostJSONRequest PostJSONRequest
func PostJSONRequest(url string, data interface{}) ([]byte, error) {
	bytesData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return PostRequest(url, "application/json;charset=UTF-8", bytes.NewReader(bytesData))
}

// GetRequest GetRequest
func GetRequest(url string) ([]byte, error) {
	client := timeoutClient()
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respbody, nil
}

// GetMD5 GetMD5
func GetMD5(str string) string {
	h := md5.New()
	io.WriteString(h, str)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// XMLDecode XMLDecode
func XMLDecode(data []byte, v interface{}) error {
	decoder := xml.NewDecoder(bytes.NewReader([]byte(data)))
	decoder.CharsetReader = charset.NewReaderLabel
	return decoder.Decode(v)
}

// Max Max
func Max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func ssrc2stream(ssrc string) string {
	if ssrc[0:1] == "0" {
		ssrc = ssrc[1:]
	}
	num, _ := strconv.Atoi(ssrc)
	return fmt.Sprintf("%X", num)
}

func sec2str(layout string, sec int64) string {
	t := time.Unix(sec, 0)
	nt := t.Format(layout)
	return nt
}

// ResolveSelfIP ResolveSelfIP
func ResolveSelfIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip, nil
		}
	}
	return nil, errors.New("server not connected to any network")
}
