package dmesg

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"
)

const (
	defaultBufSize = uint32(1 << 14)
	levelMask      = uint64(1<<3 - 1)
)

type Msg struct {
	Level      uint64
	Facility   uint64
	Seq        uint64
	TsUsec     int64
	Caller     string
	IsFragment bool
	Text       string
	DeviceInfo map[string]string
}

type dmesg struct {
	raw [][]byte
	msg []Msg
}

func parseData(data []byte) *Msg {
	msg := Msg{}

	dataLen := len(data)
	prefixEnd := bytes.IndexByte(data, ';')
	if prefixEnd == -1 {
		return nil
	}

	for index, prefix := range bytes.Split(data[:prefixEnd], []byte(",")) {
		switch index {
		case 0:
			val, _ := strconv.ParseUint(string(prefix), 10, 64)
			msg.Level = val & levelMask
			msg.Facility = val & (^levelMask)
		case 1:
			val, _ := strconv.ParseUint(string(prefix), 10, 64)
			msg.Seq = val
		case 2:
			val, _ := strconv.ParseInt(string(prefix), 10, 64)
			msg.TsUsec = val
		case 3:
			msg.IsFragment = prefix[0] != '-'
		case 4:
			msg.Caller = string(prefix)
		}
	}

	textEnd := bytes.IndexByte(data, '\n')
	if textEnd == -1 || textEnd <= prefixEnd {
		return nil
	}

	msg.Text = string(data[prefixEnd+1 : textEnd])
	if textEnd == dataLen-1 {
		return nil
	}

	msg.DeviceInfo = make(map[string]string, 2)
	deviceInfo := bytes.Split(data[textEnd+1:dataLen-1], []byte("\n"))
	for _, info := range deviceInfo {
		if info[0] != ' ' {
			continue
		}

		kv := bytes.Split(info, []byte("="))
		if len(kv) != 2 {
			continue
		}

		msg.DeviceInfo[string(kv[0])] = string(kv[1])
	}

	return &msg
}

func fetch(bufSize uint32, fetchRaw bool) (dmesg, error) {
	d := dmesg{}
	file, err := os.OpenFile("/dev/kmsg", syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return d, err
	}
	defer file.Close()

	var conn syscall.RawConn
	conn, err = file.SyscallConn()
	if err != nil {
		return d, err
	}

	if fetchRaw {
		d.raw = make([][]byte, 0)
	} else {
		d.msg = make([]Msg, 0)
	}

	truncated := false
	err = conn.Read(func(fd uintptr) bool {
		for {
			buf := make([]byte, bufSize)
			_, err := syscall.Read(int(fd), buf)
			if errors.Is(err, syscall.EINVAL) {
				truncated = true
			} else if err != nil {
				return true
			}

			if fetchRaw {
				d.raw = append(d.raw, buf)
			} else {
				msg := parseData(buf)
				if msg == nil {
					continue
				}
				d.msg = append(d.msg, *msg)
			}
		}
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "get err while fetching data from kernel:", err)
	}

	if truncated {
		err = syscall.EINVAL
	}

	return d, err
}

func DmesgWithBufSize(bufSize uint32) ([]Msg, error) {
	d, err := fetch(bufSize, false)

	return d.msg, err
}

func RawDmesgWithBufSize(bufSize uint32) ([][]byte, error) {
	d, err := fetch(bufSize, true)

	return d.raw, err
}

func Dmesg() ([]Msg, error) {
	return DmesgWithBufSize(defaultBufSize)
}

func RawDmesg() ([][]byte, error) {
	return RawDmesgWithBufSize(defaultBufSize)
}
