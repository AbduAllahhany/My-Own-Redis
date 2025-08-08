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
	Nil          = "$-1\r\n"
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
func IntegerDecoder(num int) []byte {
	return []byte(string(Integer) + strconv.Itoa(num) + CLRF)
}
func ArrayDecoder(arr []string) []byte {
	out := string(Array) + strconv.Itoa(len(arr)) + CLRF
	for _, str := range arr {
		out += string(BulkStringDecoder(str))
	}
	return []byte(out)
}
