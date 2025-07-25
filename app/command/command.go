package command

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/codecrafters-io/redis-starter-go/app/resp"
	"github.com/codecrafters-io/redis-starter-go/app/server"
	"strconv"
	"strings"
)

var (
	ErrInvalidFormat      = errors.New("invalid format")
	ErrBulkStringTooLarge = errors.New("bulk string too large")
	ErrInvalidLength      = errors.New("invalid length")
	ErrEmptyCommand       = errors.New("empty command")
)

const MaxBulkStringSize = 512 * 1024 * 1024 // 512MB limit

type HandlerCmd func(serv *server.Server, args []string) []byte

// Register command
var lookUpCommands = map[string]HandlerCmd{
	"SET":    Set,
	"GET":    Get,
	"ECHO":   Echo,
	"PING":   Ping,
	"CONFIG": Config,
	"KEYS":   Keys,
}

type Command struct {
	Name   string
	Args   []string
	handle HandlerCmd
}

func Read(reader *bufio.Reader) (Command, error) {
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
func (cmd Command) Process(serv *server.Server) []byte {
	handler := cmd.handle
	if handler == nil {
		return []byte(resp.Nil)
	}
	out := handler(serv, cmd.Args)
	return out
}

//helper function

func parseCommand(parts []string) (Command, error) {
	if len(parts) == 0 {
		return Command{}, ErrEmptyCommand
	}
	cmd := Command{
		Name:   parts[0],
		handle: lookUpCommands[strings.ToUpper(parts[0])],
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
