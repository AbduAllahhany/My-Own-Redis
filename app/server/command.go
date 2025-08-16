package server

import (
	"bufio"
	"errors"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

const MaxBulkStringSize = 512 * 1024 * 1024 // 512MB limit

var (
	ErrInvalidFormat      = errors.New("invalid format")
	ErrBulkStringTooLarge = errors.New("bulk string too large")
	ErrInvalidLength      = errors.New("invalid length")
	ErrEmptyCommand       = errors.New("empty command")
	ErrInvalidCommand     = errors.New("invalid command")
)

type HandlerCmd func(request *Request) ([]byte, error)

// Declare all maps (uninitialized)
var (
	lookUpCommands       map[string]HandlerCmd
	writeCommand         map[string]bool
	propagateCommand     map[string]bool
	suppressReplyCommand map[string]bool
)

type Command struct {
	Name           string
	Args           []string
	IsPropagatable bool
	SuppressReply  bool
	IsWritable     bool
	Handle         HandlerCmd
}

func InitCommands() {
	lookUpCommands = map[string]HandlerCmd{
		"SET":      set,
		"GET":      get,
		"ECHO":     echo,
		"PING":     ping,
		"CONFIG":   config,
		"KEYS":     keys,
		"INFO":     info,
		"PSYNC":    psync,
		"REPLCONF": replconf,
		"WAIT":     wait,
		"SELECT":   selectIndex,
	}

	writeCommand = map[string]bool{
		"SET": true,
	}

	propagateCommand = map[string]bool{
		"SET": true,
	}

	suppressReplyCommand = map[string]bool{
		"SET":      true,
		"GET":      true,
		"ECHO":     true,
		"PING":     true,
		"CONFIG":   true,
		"KEYS":     true,
		"INFO":     true,
		"PSYNC":    true,
		"REPLCONF": false,
		"SELECT":   true,
	}
}

// to be improved
func ReadCommand(reader *bufio.Reader) (Command, error) {
	//first line
	line, err := reader.ReadString('\n')
	if err != nil {
		return Command{}, err
	}
	if len(line) < 3 {
		return Command{}, ErrInvalidFormat
	}
	var parts []string
	if strings.HasSuffix(line, resp.CLRF) && line[0] == resp.Array {
		noOfParts, err := readNumbersFromLine(line)
		if err != nil {
			return Command{}, ErrInvalidLength
		}
		if noOfParts < 0 {
			return Command{}, ErrInvalidFormat
		}

		if noOfParts == 0 {
			return Command{}, ErrEmptyCommand
		}

		parts = make([]string, 0, noOfParts)
		for i := 0; i < noOfParts; i++ {
			str, err := readBulkString(reader)
			if err != nil {
				return Command{}, errors.New("failed to read bulk string")
			}
			parts = append(parts, str)
		}
	} else {
		return Command{}, ErrInvalidFormat
	}
	cmd, err := CreateCommand(parts[0], parts[1:])

	if err != nil {
		return Command{}, err
	}

	return cmd, nil
}
func ProcessCommand(request *Request) ([]byte, error) {
	handler := request.Cmd.Handle
	serv := request.Serv
	cmd := request.Cmd
	if handler == nil {
		return resp.ErrorDecoder("ERR unknown command"), ErrInvalidFormat
	}
	out, err := handler(request)
	if err != nil {
		return out, err
	}
	if serv.Role == Master && cmd.IsPropagatable && serv.ConnectedReplica != nil {
		request.Serv.Offset += len(encodeCommand(cmd))
		for _, replica := range *serv.ConnectedReplica {
			replica.Buffer.Write(encodeCommand(cmd))
			queueReplica(replica)
		}
	}
	return out, nil
}
func WriteCommand(writer *bufio.Writer, cmd *Command) error {
	result := encodeCommand(cmd)
	_, err := writer.Write(result)
	err = writer.Flush()
	return err
}

func readBulkString(reader *bufio.Reader) (string, error) {

	//first line
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", ErrInvalidFormat
	}

	if len(line) <= 0 {
		return "", ErrInvalidFormat
	}

	if strings.HasSuffix(line, resp.CLRF) && line[0] == resp.BulkString {
		noOfBytes, err := readNumbersFromLine(line)
		if err != nil {
			return "", ErrInvalidLength
		}

		if noOfBytes < 0 || noOfBytes > MaxBulkStringSize {
			return "", ErrBulkStringTooLarge
		}

		// the two bytes of CLRF
		noOfBytes += 2

		//read command
		buf := make([]byte, noOfBytes)
		n, err := reader.Read(buf)
		if n != noOfBytes {
			return "", ErrInvalidLength
		}
		if err != nil {
			return "", ErrInvalidFormat
		}
		line = string(buf)
		if !strings.HasSuffix(line, resp.CLRF) {
			return "", ErrInvalidFormat
		}
	} else {
		return "", ErrInvalidFormat
	}
	return line[:len(line)-2], nil
}
func readNumbersFromLine(line string) (int, error) {
	line = strings.TrimRight(line, resp.CLRF)
	line = line[1:]
	n, err := strconv.Atoi(line)
	if err != nil {
		return -1, err
	}
	return n, nil
}
func encodeCommand(cmd *Command) []byte {
	out := []string{cmd.Name}
	out = append(out, cmd.Args...)
	return resp.ArrayDecoder(out)
}

func CreateCommand(name string, args []string) (Command, error) {
	cmdName := strings.ToUpper(name)
	if lookUpCommands[cmdName] == nil {
		return Command{}, errors.New("Invalid Command")
	}
	return Command{
		Name:           cmdName,
		Args:           args,
		IsPropagatable: propagateCommand[cmdName],
		SuppressReply:  suppressReplyCommand[cmdName],
		IsWritable:     writeCommand[cmdName],
		Handle:         lookUpCommands[cmdName],
	}, nil
}
