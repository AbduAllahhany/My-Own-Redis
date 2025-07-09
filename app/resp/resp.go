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
	Null         = "$-1\r\n"
)

func SimpleStringDecoder(str string) []byte {
	return []byte(string(SimpleString) + str + CLRF)
}
func BulkStringDecoder(str string) []byte {
	return []byte(string(BulkString) + strconv.Itoa(len(str)) + CLRF + str + CLRF)
}
func ErrorDecoder(str string) []byte {
	return []byte(string(Error) + str + CLRF)
}
func IntegerDecoder(num string) []byte {
	return []byte(string(Integer) + num + CLRF)
}
func ArrayDecoder(arr []any) []byte {
	return nil
}
