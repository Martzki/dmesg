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
	levelMask      = uint64(1 << 3 - 1)
)

type msgElem struct {
	Level      uint64
	Facility   uint64
	Seq        uint64
	TsUsec     int64
	Caller     string
	IsFragment bool
	Text       string
	DeviceInfo map[string]string
	Raw        []byte
}

type Dmesg struct {
	num          uint64
	msgTruncated bool
	bufSize      uint32
	msg          []*msgElem
}

func parseData(data []byte) *msgElem {
	msg := msgElem{}
	msg.Raw = data

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

func parseMsg(msg *msgElem) string {
	res := fmt.Sprintf("<%d>[%d][%d][%d][%s][%t] %s", msg.Level, msg.Facility,
		msg.Seq, msg.TsUsec, msg.Caller, msg.IsFragment, msg.Text)
	for k, v := range msg.DeviceInfo {
		res += fmt.Sprintf("\n %s=%s", k, v)
	}

	return res
}

func (d *Dmesg) SetBufSize(size uint32) {
	d.bufSize = size
}

func (d *Dmesg) fetchRawMsg() {
	file, err := os.OpenFile("/dev/kmsg", syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open /dev/kmsg:", err)
		return
	}
	defer file.Close()

	var conn syscall.RawConn
	conn, err = file.SyscallConn()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to get raw fd:", err)
		return
	}

	err = conn.Read(func(fd uintptr) bool {
		d.msgTruncated = false
		num := uint64(0)
		for {
			buf := make([]byte, d.bufSize)
			_, err := syscall.Read(int(fd), buf)
			if errors.Is(err, syscall.EINVAL) {
				d.msgTruncated = true
			} else if err != nil {
				d.num = num
				return true
			}

			msg := parseData(buf)
			if msg == nil {
				continue
			}

			d.msg = append(d.msg, msg)
			num++
		}
	})

	if err != nil {
		fmt.Fprintln(os.Stderr, "get err while fetching data from kernel:", err)
	}
}

func (d *Dmesg) FetchText(num uint64, reverse bool) ([]string, error) {
	n := min(d.num, num)
	res := make([]string, n)
	for i := uint64(0); i < n; i++ {
		res[i] = parseMsg(d.msg[i])
	}

	if d.msgTruncated {
		return res, syscall.ENOBUFS
	}

	return res, nil
}

func (d *Dmesg) FetchTextAll(reverse bool) ([]string, error) {
	return d.FetchText(d.num, reverse)
}

func (d *Dmesg) FetchRaw() ([]*msgElem, error) {
	if d.msgTruncated {
		return d.msg, syscall.ENOBUFS
	} else {
		return d.msg, nil
	}
}

func NewDmesg() *Dmesg {
	d := &Dmesg{}
	d.bufSize = defaultBufSize
	d.fetchRawMsg()

	return d
}
