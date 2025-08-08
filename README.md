# Redis Clone Implementation in Go

A Redis server implementation built in Go as part of the . This implementation supports essential Redis commands, master-slave replication, and follows the Redis Serialization Protocol (RESP).

## Features

### Core Commands

- **PING** - Test server connectivity
- **ECHO** - Echo back a message
- **GET** - Retrieve a value by key
- **SET** - Store a key-value pair with optional expiration
- **CONFIG GET** - Get configuration values
- **KEYS** - Find keys matching a pattern

### Replication Commands

- **INFO** - Get server information (replication status)
- **REPLCONF** - Replication configuration
- **PSYNC** - Partial synchronization for replication
- **WAIT** - Wait for replica acknowledgments

### Advanced Features

- **Master-Slave Replication** - Full master-slave replication support
- **TTL Support** - Keys can expire after a specified time
- **RESP Protocol** - Full Redis Serialization Protocol implementation
- **Concurrent Connections** - Multi-client support with goroutines
- **RDB File Loading** - Load initial data from Redis RDB files
- **Thread-Safe Operations** - Concurrent read/write operations with proper locking
- **Command Propagation** - Commands are propagated to replicas

## Architecture

### Core Components

```
app/
├── main.go              # Server entry point and connection handling
├── engine/              # Data storage engine
│   └── engine.go        # In-memory database with TTL support
├── resp/                # Redis Serialization Protocol
│   └── resp.go          # RESP encoding/decoding
├── rdb/                 # RDB file support
│   └── rdb.go           # RDB file parsing and generation
├── server/              # Server core functionality
│   ├── server.go        # Server setup, configuration, and replication
│   ├── command.go       # Command parsing and processing logic
│   ├── server_command.go # Server-level commands (PING, ECHO, CONFIG, etc.)
│   └── string_command.go # String operations (GET, SET)
└── utils/               # Utility functions
    └── utils.go         # ID generation and helpers
```

### Key Design Decisions

1. **Modular Architecture** - Separated concerns into distinct packages
2. **Interface-Based Design** - `RedisObj` interface allows for different data types
3. **Thread Safety** - Uses `sync.RWMutex` for concurrent access
4. **Memory Management** - Efficient in-memory storage with expiration cleanup
5. **Protocol Compliance** - Full RESP implementation for Redis compatibility
6. **Replication Support** - Complete master-slave replication with command propagation

## Quick Start

### Prerequisites

- Go 1.24 or higher
- Git

### Installation

```bash
git clone https://github.com/AbduAllahhany/My-Own-Redis
cd My-Own-Redis
```

### Running the Server

#### Master Mode (Default)
```bash
# Quick start
./your_program.sh
```


#### Slave Mode
```bash
# Connect to a master running on localhost:6380
./your_program.sh --port 6379 --replicaof "localhost 6380"
```

### Quick Test

```bash
# Connect to the server
redis-cli -p 6379

# Test basic functionality
127.0.0.1:6379> PING
PONG

127.0.0.1:6379> SET hello "world"
OK

127.0.0.1:6379> GET hello
"world"
```

## Configuration

The server accepts the following command-line flags:

- `--dir` - Directory path for data storage (default: `/tmp`)
- `--dbfilename` - Database filename (default: `dump.rdb`)
- `--port` - Server port (default: `6379`)
- `--replicaof` - Master server for replication (format: "host port")

### Example Configurations

```bash
# Master with custom data directory
./redis-server --dir /var/redis --dbfilename production.rdb --port 6379

# Slave connecting to remote master
./redis-server --port 6380 --replicaof "192.168.1.100 6379"
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

# Test replication info
127.0.0.1:6379> INFO replication
# REPLICATION
role:master
master_replid:abc123...
master_repl_offset:0
```

## Command Reference

### Basic Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| PING | `PING` | Returns PONG |
| ECHO | `ECHO message` | Returns the message |
| GET | `GET key` | Get value of key |
| SET | `SET key value [EX seconds] [PX milliseconds] [GET]` | Set key to value with optional expiration |
| KEYS | `KEYS pattern` | Find keys matching pattern |
| CONFIG GET | `CONFIG GET parameter` | Get configuration parameter |

### Replication Commands

| Command | Syntax | Description |
|---------|--------|-------------|
| INFO | `INFO replication` | Get replication information |
| REPLCONF | `REPLCONF option value` | Configure replication |
| PSYNC | `PSYNC replicationid offset` | Initiate partial sync |
| WAIT | `WAIT numreplicas timeout` | Wait for replica acknowledgments |

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

## Replication

This implementation supports Redis-compatible master-slave replication:

### Master Setup
```bash
# Master runs on default port 6379
./redis-server
```

### Slave Setup
```bash
# Slave connects to master
./redis-server --port 6380 --replicaof "localhost 6379"
```

### Replication Features
- **Full Synchronization** - New replicas receive a complete RDB snapshot
- **Command Propagation** - Write commands are propagated to all replicas
- **Replica Acknowledgments** - Support for WAIT command to ensure replica consistency
- **Automatic Reconnection** - Slaves automatically reconnect to masters

### Testing Master-Slave Replication

```bash
# Terminal 1: Start master
./redis-server --port 6379

# Terminal 2: Start slave
./redis-server --port 6380 --replicaof "localhost 6379"

# Terminal 3: Test replication
redis-cli -p 6379
127.0.0.1:6379> SET test "hello"
OK
127.0.0.1:6379> GET test
"hello"

# Terminal 4: Verify on slave
redis-cli -p 6380
127.0.0.1:6380> GET test
"hello"
```

## Protocol Details

This implementation follows the Redis Serialization Protocol (RESP):

- **Simple Strings** - `+OK\r\n`
- **Errors** - `-ERR message\r\n`
- **Integers** - `:1000\r\n`
- **Bulk Strings** - `$6\r\nfoobar\r\n`
- **Arrays** - `*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n`
- **Null Bulk String** - `$-1\r\n`

## Development

### Project Structure

The codebase is organized into logical packages:

- **engine** - Data storage and retrieval with TTL support
- **resp** - Protocol encoding/decoding
- **rdb** - RDB file format support for persistence
- **server** - Server configuration, networking, and replication
- **utils** - Utility functions

### Adding New Commands

1. Define the command handler in the appropriate command file
2. Register it in the `lookUpCommands` map in `server/command.go`
3. Configure command properties in `writeCommand`, `propagateCommand`, and `suppressReplyCommand` maps
4. Implement the command logic following the established patterns

### Error Handling

The implementation includes comprehensive error handling:

- Protocol parsing errors
- Command syntax errors
- Type errors (e.g., operating on wrong data types)
- Resource limits (bulk string size limits)
- Connection errors and timeouts

## Performance

### Performance Considerations

- **Concurrent Access** - Uses read-write mutexes for optimal performance
- **Memory Efficiency** - In-memory storage with automatic expiration
- **Connection Pooling** - Supports multiple concurrent client connections
- **Bulk Operations** - Efficient bulk string handling with size limits
- **Non-blocking Replication** - Background RDB generation and command buffering

## Limitations

Current limitations compared to full Redis:

- **Limited Data Types** - Only strings currently implemented (lists partially defined)
- **No Persistence** - Data is lost on restart (except initial RDB loading)
- **No Clustering** - Single master-slave replication only
- **No Pub/Sub** - No publish/subscribe functionality
- **No Transactions** - No MULTI/EXEC support
- **No Lua Scripting** - No embedded Lua interpreter
- **Basic Pattern Matching** - KEYS command uses simple glob patterns

## TODO

Based on the code comments, planned improvements include:

1. **Testing** - Comprehensive test suite
2. **Custom RDB Reader** - Replace third-party RDB library
3. **Memory Optimization** - Improve memory usage patterns
4. **Connection Pool** - Better connection management
5. **Logging** - Structured logging system
6. **Offset Tracking** - Improved replication offset handling
7. **Command Processing** - Enhanced readCommand function
8. **I/O Abstraction** - Server-level reader/writer interfaces

