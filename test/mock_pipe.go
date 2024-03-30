// Code generated by interfacer; DO NOT EDIT

package mock

import (
	"github.com/bitfield/script"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Pipe is an interface generated for "github.com/bitfield/script.Pipe".
type Pipe interface {
	AppendFile(string) (int64, error)
	Basename() *script.Pipe
	Bytes() ([]byte, error)
	Close() error
	Column(int) *script.Pipe
	Concat() *script.Pipe
	CountLines() (int, error)
	Dirname() *script.Pipe
	Do(*http.Request) *script.Pipe
	EachLine(func(string, *strings.Builder)) *script.Pipe
	Echo(string) *script.Pipe
	Error() error
	Exec(string) *script.Pipe
	ExecForEach(string) *script.Pipe
	ExitStatus() int
	Filter(func(io.Reader, io.Writer) error) *script.Pipe
	FilterLine(func(string) string) *script.Pipe
	FilterScan(func(string, io.Writer)) *script.Pipe
	First(int) *script.Pipe
	Freq() *script.Pipe
	Get(string) *script.Pipe
	JQ(string) *script.Pipe
	Join() *script.Pipe
	Last(int) *script.Pipe
	Match(string) *script.Pipe
	MatchRegexp(*regexp.Regexp) *script.Pipe
	Post(string) *script.Pipe
	Read([]byte) (int, error)
	Reject(string) *script.Pipe
	RejectRegexp(*regexp.Regexp) *script.Pipe
	Replace(string, string) *script.Pipe
	ReplaceRegexp(*regexp.Regexp, string) *script.Pipe
	SHA256Sum() (string, error)
	SHA256Sums() *script.Pipe
	SetError(error)
	Slice() ([]string, error)
	Stdout() (int, error)
	String() (string, error)
	Tee(...io.Writer) *script.Pipe
	Wait()
	WithError(error) *script.Pipe
	WithHTTPClient(*http.Client) *script.Pipe
	WithReader(io.Reader) *script.Pipe
	WithStderr(io.Writer) *script.Pipe
	WithStdout(io.Writer) *script.Pipe
	WriteFile(string) (int64, error)
}
