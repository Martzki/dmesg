# dmesg
Golang package to get message from Linux kernel ring buffer.
This package gets message by reading from `/dev/kmsg` like cmd util `dmesg`.

# types
## Msg
```go
type Msg struct {
	Level      uint64            // SYSLOG lvel
	Facility   uint64            // SYSLOG facility
	Seq        uint64            // Message sequence number
	TsUsec     int64             // Timestamp in microsecond
	Caller     string            // Message caller
	IsFragment bool              // This message is a fragment of an early message which is not a fragment
	Text       string            // Log text
	DeviceInfo map[string]string // Device info
}
```
`Msg` is a serialized message structure by parsing native message. It returned by `Dmesg` or `DmesgWithBufSize`.

# functions
## Dmesg
```go
func Dmesg() ([]Msg, error)
```
Dmesg gets all messages from kernel ring buffer with default buf size 16KB for each message.  
It returns serialized message structure and the error while getting messages.  
The error `syscall.EINVAL` means the buf size is not enough, consider to use `DmesgWithBufSize` instead.  
## RawDmesg
```go
func RawDmesg() ([][]byteg, error)
```
RawDmesg gets all messages from kernel ring buffer with default buf size 16KB for each message.  
It returns native message from kernel without parsing and the error while getting messages.  
The error `syscall.EINVAL` means the buf size is not enough, consider to use `RawDmesgWithBufSize` instead.
## DmesgWithBufSize
```go
func DmesgWithBufSize(bufSize uint32) ([]Msg, error)
```
DmesgWithBufSize gets all messages from kernel ring buffer with specific buf size for each message.  
It returns serialized message structure and the error while getting messages.
## RawDmesgWithBufSize
```go
func RawDmesgWithBufSize(bufSize uint32) ([][]byte, error)
```
RawDmesgWithBufSize gets all messages from kernel ring buffer with specific buf size for each message.  
It returns native message from kernel without parsing and the error while getting messages.