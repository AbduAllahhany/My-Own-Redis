# Redis Clone Implementation in Go

A toy Redis server implementation built in Go as part of the [CodeCrafters Redis Challenge](https://codecrafters.io/challenges/redis). This implementation supports basic Redis commands and follows the Redis Serialization Protocol (RESP).

## Features

### Supported Commands

- **PING** - Test server connectivity
- **ECHO** - Echo back a message
- **GET** - Retrieve a value by key
- **SET** - Store a key-value pair with optional expiration
- **CONFIG GET** - Get configuration values
- **KEYS** - Find keys matching a pattern

### Advanced Features

- **TTL Support** - Keys can expire after a specified time
- **RESP Protocol** - Full Redis Serialization Protocol implementation
- **Concurrent Connections** - Multi-client support with goroutines
- **RDB File Loading** - Load initial data from Redis RDB files
- **Thread-Safe Operations** - Concurrent read/write operations with proper locking

## Architecture

### Core Components

```
app/
├── main.go              # Server entry point and connection handling
├── command/             # Command parsing and execution
│   ├── command.go       # Core command parsing logic
│   ├── server_command.go # Server-level commands (PING, ECHO, CONFIG, KEYS)
│   └── string_command.go # String operations (GET, SET)
├── engine/              # Data storage engine
│   └── engine.go        # In-memory database with TTL support
├── resp/                # Redis Serialization Protocol
│   └── resp.go          # RESP encoding/decoding
├── rdb/                 # RDB file support
│   └── rdb.go           # RDB file parsing
└── server/              # Server configuration and initialization
    └── server.go        # Server setup and configuration
```

### Key Design Decisions

1. **Modular Architecture** - Separated concerns into distinct packages
2. **Interface-Based Design** - `RedisObj` interface allows for different data types
3. **Thread Safety** - Uses `sync.RWMutex` for concurrent access
4. **Memory Management** - Efficient in-memory storage with expiration cleanup
5. **Protocol Compliance** - Full RESP implementation for Redis compatibility

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Git

### Installation

```bash
git clone <repository-url>
cd redis-starter-go
```

### Running the Server

```bash
# Local development
./your_program.sh

# Or build and run manually
go build -o redis-server app/*.go
./redis-server
```

### Configuration Options

The server accepts the following command-line flags:

- `--dir` - Directory path for data storage (default: `/tmp`)
- `--dbfilename` - Database filename (default: `dump.rdb`)

Example:
```bash
./redis-server --dir /data --dbfilename my_database.rdb
```

### Testing with Redis CLI

```bash
# Connect to the server
redis-cli -p 6379

# Test basic commands
127.0.0.1:6379> PING
PONG

127.0.0.1:6379> SET mykey "Hello World"
OK

127.0.0.1:6379> GET mykey
"Hello World"

127.0.0.1:6379> SET temp "value" EX 10
OK

127.0.0.1:6379> KEYS *
1) "mykey"
2) "temp"
```

## Command Reference

### Basic Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| PING | `PING` | Returns PONG |
| ECHO | `ECHO message` | Returns the message |
| GET | `GET key` | Get value of key |
| SET | `SET key value [EX seconds] [PX milliseconds]` | Set key to value with optional expiration |
| KEYS | `KEYS pattern` | Find keys matching pattern |
| CONFIG GET | `CONFIG GET parameter` | Get configuration parameter |

### SET Command Options

- `EX seconds` - Set expiration in seconds
- `PX milliseconds` - Set expiration in milliseconds  
- `GET` - Return the old value (if any)

Examples:
```bash
SET mykey "value" EX 30        # Expires in 30 seconds
SET mykey "value" PX 5000      # Expires in 5000 milliseconds
SET mykey "newvalue" GET       # Returns old value, sets new value
```

## Protocol Details

This implementation follows the Redis Serialization Protocol (RESP):

- **Simple Strings** - `+OK\r\n`
- **Errors** - `-ERR message\r\n`
- **Integers** - `:1000\r\n`
- **Bulk Strings** - `$6\r\nfoobar\r\n`
- **Arrays** - `*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n`

## Development

### Project Structure

The codebase is organized into logical packages:

- **command** - Command parsing and execution
- **engine** - Data storage and retrieval
- **resp** - Protocol encoding/decoding
- **rdb** - RDB file format support
- **server** - Server configuration and lifecycle

### Adding New Commands

1. Define the command handler in the appropriate command file
2. Register it in the `lookUpCommands` map in `command/command.go`
3. Implement the command logic following the established patterns

### Error Handling

The implementation includes comprehensive error handling:

- Protocol parsing errors
- Command syntax errors
- Type errors (e.g., operating on wrong data types)
- Resource limits (bulk string size limits)

## Performance Considerations

- **Concurrent Access** - Uses read-write mutexes for optimal performance
- **Memory Efficiency** - In-memory storage with automatic expiration
- **Connection Pooling** - Supports multiple concurrent client connections
- **Bulk Operations** - Efficient bulk string handling with size limits

## Limitations

Current limitations compared to full Redis:

- Limited data types (only strings currently implemented)
- No persistence (data is lost on restart, except RDB loading)
- No clustering or replication
- No pub/sub functionality
- No transactions
- No Lua scripting


