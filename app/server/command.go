package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/codecrafters-io/redis-starter-go/app/resp"
)

var (
	ErrInvalidFormat      = errors.New("invalid format")
	ErrBulkStringTooLarge = errors.New("bulk string too large")
	ErrInvalidLength      = errors.New("invalid length")
	ErrEmptyCommand       = errors.New("empty command")
)

const MaxBulkStringSize = 512 * 1024 * 1024 // 512MB limit

type HandlerCmd func(request *Request) []byte

// Register command
var lookUpCommands = map[string]HandlerCmd{
	"SET":      Set,
	"GET":      Get,
	"ECHO":     Echo,
	"PING":     Ping,
	"CONFIG":   Config,
	"KEYS":     Keys,
	"INFO":     Info,
	"PSYNC":    Psync,
	"REPLCONF": Replconfg,
}
var writeCommand = map[string]bool{
	"SET": true,
}

type Command struct {
	Name       string
	Args       []string
	IsWritable bool
	Handle     HandlerCmd
}

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
				return Command{}, fmt.Errorf("failed to read bulk string %d: %w", i, err)
			}
			parts = append(parts, str)
		}
	} else {
		return Command{}, ErrInvalidFormat
	}
	cmd, err := parseCommand(parts)

	if err != nil {
		return Command{}, err
	}

	return cmd, nil
}
func (serv *Server) ProcessCommand(conn *net.Conn, cmd *Command) []byte {
	handler := cmd.Handle
	if handler == nil {
		return resp.ErrorDecoder("ERR unknown command")
	}
	req := Request{
		Serv: serv,
		Args: cmd.Args,
		Conn: conn,
	}
	out := handler(&req)
	if cmd.IsWritable {
		for _, replica := range serv.ConnectedReplica {
			replicaConn := *replica.Conn
			writer := bufio.NewWriter(replicaConn)
			WriteCommand(writer, cmd)
		}
	}
	return out
}

// helper function
func parseCommand(parts []string) (Command, error) {
	if len(parts) == 0 {
		return Command{}, ErrEmptyCommand
	}
	cmdName := strings.ToUpper(parts[0])
	cmd := Command{
		Name:       cmdName,
		Handle:     lookUpCommands[strings.ToUpper(parts[0])],
		IsWritable: writeCommand[cmdName],
	}
	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}
	return cmd, nil
}

func readBulkString(reader *bufio.Reader) (string, error) {

	//first line
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println(err)
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

func WriteCommand(writer *bufio.Writer, cmd *Command) error {
	out := []string{cmd.Name}
	out = append(out, cmd.Args...)
	result := resp.ArrayDecoder(out)
	_, err := writer.Write(result)
	err = writer.Flush()
	return err
}
