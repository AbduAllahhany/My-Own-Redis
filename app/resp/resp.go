package resp

import (
	"strconv"
)

const (
	SimpleString = '+'
	Error        = '-'
	Integer      = ':'
	BulkString   = '$'
	Array        = '*'
	CLRF         = "\r\n"
)

type Decoder interface {
}

type SimpleStringDecoder struct {
}
type BulkStringDecoder struct {
}
type ErrorDecoder struct {
}
type IntegerDecoder struct {
}
type ArrayDecoder struct {
}

func (d SimpleStringDecoder) Decode(str string) string {
	return string(SimpleString) + str + CLRF
}
func (d BulkStringDecoder) Decode(str string) string {
	return string(BulkString) + strconv.Itoa(len(str)) + CLRF + str + CLRF
}

func (d ErrorDecoder) Decode(str string) string {
	return string(Error) + str + CLRF
}
func (d IntegerDecoder) Decode(num int) string {
	return string(Integer) + strconv.Itoa(num) + CLRF
}

func (d ArrayDecoder) Decode(arr []any) string {
	return ""
}
